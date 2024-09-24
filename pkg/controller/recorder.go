// Copyright (c) stagger team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package controller

import (
	"stagger/pkg/controller/types"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

type Recorder struct {
	recorder record.EventRecorder
	object   runtime.Object
}

var _ types.ObjectRecorder = &Recorder{}

func NewRecorderForObject(recorder record.EventRecorder, object runtime.Object) *Recorder {
	return &Recorder{
		recorder: recorder,
		object:   object,
	}
}

func (r *Recorder) Normalf(reason, format string, args ...interface{}) {
	r.recorder.Eventf(r.object, corev1.EventTypeNormal, reason, format, args...)
}

func (r *Recorder) Warnf(reason, format string, args ...interface{}) {
	r.recorder.Eventf(r.object, corev1.EventTypeWarning, reason, format, args...)
}

func (r *Recorder) Logf(logger logr.Logger, v int, reason, format string, args ...interface{}) {
	/*
		 TODO: do we need this?
			logger.Logf(level, format, args...)

			switch level {
			case logging.TraceLevel:
			case logging.DebugLevel:
			case logging.InfoLevel:
				r.Normalf(reason, format, args...)
			case logging.WarnLevel:
				fallthrough
			case logging.ErrorLevel:
				r.Warnf(reason, format, args...)
			}
	*/
}
