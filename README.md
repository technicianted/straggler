Stagger - A pod staggering controller
---

Stagger is an in-cluster controller that can be used to stagger starting of pods across multiple namespaces and controllers in order to control thundering herd effects in large-scale systems.

A typical scenario is where large number of pods need to be created rapidly. This typically puts pressure on resources as all of them rush to do things like image pulls, data downloads and even pressure the API server.

Stagger provides multiple staggering policies. Policies define matching pods, a grouping key and a pacer. Pods are matched using label selectors. A grouping key is a jsonpath expression that gets applied to pod specs. They key places all pods in a single staggering group. Finally, each staggering group has a defined pacer that controls how fast the pods are started.

### Example
Consider the following example where we want to stagger access to image pulls such that something like [spegel](https://github.com/spegel-org/spegel) gets a chance to seed the images. We want to control staggering per image, not as a whole for cache population and seeding:
```yaml
staggeringPolicies:
# create a staggering policy to control image pulls.
- name: image-pull
  # only pods carrying this label will be considered.
  labelSelector:
    staggerimages: "1"
  # stagger pods in groups by evaluating this jsonpath.
  # in other words, pods with similar images are put in
  # the same group.
  groupingExpression: .spec.containers[0].image
  pacer:
    # use exponential pacer. start with 4 initially then
    # go to 16, then finally start all remaining pods.
    exponential:
      minInitial: 4
      maxStagger: 16
      multiplier: 4
```

### How it works
Stagger works by monitoring pods via an admission controller. With each new pods, it is evaluated against defined policies. Once it is associated with one, its pacer is consulted to see if it should be allowed to start. If it is not, a special `nodeSelector` is added to block its scheduling.

Next, a reconciler controller monitors pods events and status changes. With each change of a staggered pod, its corresponding pacer is consulted. If it is allowed to start, the pod is evicted and will be recreated where the admission controller will let it be scheduled.

### Job controllers special handling
Special handling is needed for pods created by a Job controller. By default, Job controllers do not differentiate between an evicted pod and a failed one. Since we use pod eviction to reschedule the pod, Job specs need to be changed to inject the following failure policy:
```yaml
  podFailurePolicy:
    rules:
    - action: Ignore
      onPodConditions:
      - type: DisruptionTarget
```

**Note: if your Job spec already has a `DisruptionTarget` policy with `action` not set to `Ignore`, stagger will issue a warning and will not apply policies**
