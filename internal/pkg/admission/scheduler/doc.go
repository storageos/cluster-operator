// Package scheduler contains mutating admission controller webhook handlers for
// pod scheduler. The handler mutates a given pod if the pod has any volume
// that's managed by the given volume provisioners in
// PodSchedulerSetter.Provisioners and the associated scheduler is enabled.
// The scheduler name is set in PodSchedulerSetter.SchedulerName.
package scheduler
