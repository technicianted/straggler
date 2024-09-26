// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package controller

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type flight struct {
	object    client.ObjectKey
	timestamp time.Time
}

type flightList struct {
	flights    *list.List
	landedChan chan interface{}
	// maintain a list by grouping key to accomodate for
	// potential changes
	seenPodsByNamespacedName map[apitypes.NamespacedName]bool
}

type flightTracker struct {
	sync.Mutex

	client                       client.Client
	maxFlightDuration            time.Duration
	objectKeyLabel               string
	flightsByKey                 map[string]*flightList
	seenPodsKeysByNamespacedName map[apitypes.NamespacedName]string
}

// Create a new flight tracker for pods between admission until commitment in
// API server.
// This implemnetation is best effort. There are many race conditions that are
// unavoidable but we try to minimize them.
// Implementation uses client to fetch and check committed objects. For safety,
// pods that exceed maxFlightDuration and automatically assumed committed.
// objectKeyLabel should match the same label that the admision controller
// uses to set the grouping key.
//
// TODO: This is overly complex.
func NewFlightTracker(
	client client.Client,
	maxFlightDuration time.Duration,
	objectKeyLabel string,
	logger logr.Logger,
) *flightTracker {
	tracker := &flightTracker{
		client:                       client,
		maxFlightDuration:            maxFlightDuration,
		objectKeyLabel:               objectKeyLabel,
		flightsByKey:                 make(map[string]*flightList),
		seenPodsKeysByNamespacedName: make(map[apitypes.NamespacedName]string),
	}
	tracker.startEvicter(logger)

	return tracker
}

// Track a pod until it get recinciled in the API server. This includes pods that
// don't yet have a Name but only GenerateName.
func (f *flightTracker) Track(key string, object metav1.ObjectMeta, logger logr.Logger) error {
	f.Lock()
	defer f.Unlock()

	name := object.Name
	if len(name) == 0 {
		name = object.GenerateName
	}
	if len(name) == 0 {
		return fmt.Errorf("unable to get a unique name for object")
	}

	flights, ok := f.flightsByKey[key]
	if !ok {
		flights = &flightList{
			flights:                  list.New(),
			landedChan:               make(chan interface{}),
			seenPodsByNamespacedName: make(map[apitypes.NamespacedName]bool),
		}
		f.flightsByKey[key] = flights
	}

	flightName := apitypes.NamespacedName{
		Name:      name,
		Namespace: object.Namespace,
	}
	flights.flights.PushBack(&flight{
		object:    flightName,
		timestamp: time.Now(),
	})

	logger.Info("tracking new flight", "flight", flightName, "key", key)

	return nil
}

// Waits for one flight to land with key or ctx timeout.
func (f *flightTracker) WaitOne(ctx context.Context, key string, logger logr.Logger) error {
	f.Lock()

	flights, ok := f.flightsByKey[key]
	if !ok {
		logger.V(1).Info("flight key entry does not exist", "key", key)
		f.Unlock()
		return nil
	}
	waitChan := flights.landedChan
	f.Unlock()

	logger.V(10).Info("awaiting a flight to land", "key", key)
	select {
	case <-waitChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Reconciles committed pods and landing corresponding pending entries. Note that in cases where
// a pod do not have a name yet, we try to match any with the same GenerateName.
func (f *flightTracker) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := logf.FromContext(ctx)

	f.Lock()
	defer f.Unlock()

	pod := &corev1.Pod{}
	err := f.client.Get(ctx, request.NamespacedName, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			if key, ok := f.seenPodsKeysByNamespacedName[request.NamespacedName]; ok {
				if flightList, ok := f.flightsByKey[key]; ok {
					delete(flightList.seenPodsByNamespacedName, request.NamespacedName)
				}
			}
			return reconcile.Result{}, nil
		} else {
			logger.Error(err, "failed to get Pod")
			return reconcile.Result{}, err
		}
	}

	key, ok := pod.Labels[f.objectKeyLabel]
	if !ok {
		logger.Info("got pod with no key label", "label", f.objectKeyLabel, "pod", request.NamespacedName)
		return reconcile.Result{}, nil
	}

	flightList, ok := f.flightsByKey[key]
	if !ok {
		logger.V(1).Info("flight key entry does not exist", "key", key)
		return reconcile.Result{}, nil
	}

	if _, ok := flightList.seenPodsByNamespacedName[request.NamespacedName]; ok {
		logger.V(10).Info("skipping seen pod")
		return reconcile.Result{}, nil
	}
	matched := false
	// look for matching flight.
	// a matching flight can be name+namespace or generatedName+namepsace
	if len(pod.Name) != 0 {
		// try first full name match
		for el := flightList.flights.Back(); el != nil; el = el.Prev() {
			flight := el.Value.(*flight)
			if flight.object.Name == pod.Name &&
				flight.object.Namespace == pod.Namespace {
				matched = true
				logger.V(1).Info("found exact name match", "key", key, "flight", flight)
				flightList.flights.Remove(el)
				break
			}
		}
	}
	if !matched {
		for el := flightList.flights.Back(); el != nil; el = el.Prev() {
			flight := el.Value.(*flight)
			if flight.object.Name == pod.GenerateName &&
				flight.object.Namespace == pod.Namespace {
				matched = true
				logger.V(1).Info("found generate name match", "key", key, "flight", flight)
				flightList.flights.Remove(el)
				break
			}
		}
	}
	if matched {
		flightList.seenPodsByNamespacedName[request.NamespacedName] = true
		f.seenPodsKeysByNamespacedName[request.NamespacedName] = key
		go func() {
			flightList.landedChan <- nil
		}()
	} else {
		logger.V(10).Info("no matched flight found")
	}

	return reconcile.Result{}, nil
}

func (f *flightTracker) startEvicter(logger logr.Logger) {
	logger = logger.WithName("tracker")
	logger.Info("starting flight tracker")

	go func() {
		for {
			<-time.After(f.maxFlightDuration / 2)
			f.Lock()

			flightLists := map[string]*flightList{}
			for key := range f.flightsByKey {
				flightList := f.flightsByKey[key]
				if flightList.flights.Len() == 0 &&
					len(flightList.seenPodsByNamespacedName) == 0 {
					delete(f.flightsByKey, key)
				} else {
					flightLists[key] = f.flightsByKey[key]
				}
			}

			for key := range flightLists {
				flightList := flightLists[key]
				// loop from oldest to newest
				el := flightList.flights.Back()
				for el != nil {
					flight := el.Value.(*flight)
					if time.Since(flight.timestamp) > f.maxFlightDuration {
						logger.Info("force landing flight", "flight", flight.object)
						go func() {
							flightList.landedChan <- nil
						}()
						next := el.Prev()
						flightList.flights.Remove(el)
						el = next
					} else {
						el = el.Prev()
					}
				}
			}

			f.Unlock()
		}
	}()
}
