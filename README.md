Straggler - A pod staggering controller
---

Straggler is an in-cluster controller that can be used to straggler starting of pods across multiple namespaces and controllers in order to control thundering herd effects in large-scale systems.

A typical scenario is where large number of pods need to be created rapidly. This typically puts pressure on resources as all of them rush to do things like image pulls, data downloads and even pressure the API server.

Straggler provides multiple staggering policies. Policies define matching pods, a grouping key and a pacer. Pods are matched using label selectors. A grouping key is a jsonpath expression that gets applied to pod specs. They key places all pods in a single staggering group. Finally, each staggering group has a defined pacer that controls how fast the pods are started.

### Building

```bash
$ # build binaries only in bin/straggler
$ make binaries
$ # build docker image
$ make docker-build
```

### Running
Simplest way is to use the helm chart. Edit the file `configs/_policies.yaml` to define your staggering policies then install the helm chart:
```bash
$ helm upgrade \
  --install \
  --namespace straggler \
  --create-namespace 
  straggler helm/straggler
```

### Example
Consider the following example where we want to straggler access to image pulls such that something like [spegel](https://github.com/spegel-org/spegel) gets a chance to seed the images. We want to control staggering per image, not as a whole for cache population and seeding:
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

then configure your pods with the labels:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
        # enable staggering for pods created by this controller.
        v1.straggler.technicianted/enable: "1"
        # enable image staggering policy.
        staggerimages: "1"
    spec:
      containers:
      - name: nginx
        # all pods for this deployment and any other controllers 
        # using this image will share the same policy and staggering group.
        image: nginx:1.14.2
```

### Staggering bypass

In some situations where a staggering policy spans multiple pods controlled by different Kubernets controllers, we may want to bypass staggering for a certain set of these pods due to subtle startup dependencies. To do that, policies include `BypassLabelSelector` that lets you specify a label selector that if matched, this policy will not apply but the pod itself will be counted against pacing.

### Job controllers special handling
Special handling is needed for pods created by a Job controller. By default, Job controllers do not differentiate between an evicted pod and a failed one. Since we use pod eviction to reschedule the pod, Job specs need to be changed to inject the following failure policy:
```yaml
  podFailurePolicy:
    rules:
    - action: Ignore
      onPodConditions:
      - type: DisruptionTarget
```

**Note: if your Job spec already has a `DisruptionTarget` policy with `action` not set to `Ignore`, straggler will issue a warning and will not apply policies**

### FAQ
* **Can a single straggler group span multiple controllers?**

Yes. It can even span multiple namespaces.

* **How do I know if a pod is being staggered?**

If a pod is being staggered awaiting pacing, it will have label `v1.straggler.technicianted/staggered=1` set. You can list these pods using something like:
```bash
$ kubectl get pods -l v1.straggler.technicianted/staggered=1
```

* **How are pods prevented from starting up (staggered)?**

Straggler works by monitoring pods via an admission controller. With each new pods, it is evaluated against defined policies. Once it is associated with one, its pacer is consulted to see if it should be allowed to start. If it is not, a special pod specs are replaced with stub specs with same resources. Further, an init container is appended that will block the startup of the pod. When the reconciler is ready, the pod is evicted and restarted with its original specs.

Next, a reconciler controller monitors pods events and status changes. With each change of a staggered pod, its corresponding pacer is consulted. If it is allowed to start, the pod is evicted and will be recreated where the admission controller will let it be scheduled.

* **Why not use `scale` subresource?**

One of the important design objectives is to be controller agnostic, and be able to straggler across multiple controllers. If `scale` subresource is used as a mechanism of staggering then it'll pose many restrictions. For example, the owning controller must support `scale`. Also other controllers such as HPA may be already controlling the `scale` subresource and will conflict with staggering.

* **What about gang scheduling?**

Gang scheduling blocks the scheduling of a group of pods until all requested resources are ready. Straggler handles this by using an init container to block the startup of the pod such that gang schedulers are not affected.