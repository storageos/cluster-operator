// Package cert is a copy of client-go's util/cert package from k8s 1.13. In
// k8s 1.14, the cert utilities were moved into kubeadm package and are no
// longer available via any importable library. This is used by the admission
// controller to generate a self signed certificate in the operator.
package cert
