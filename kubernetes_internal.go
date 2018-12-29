package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	cni "github.com/K8sNetworkPlumbingWG/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/ericchiang/k8s"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
)

type netAttachDef cni.NetworkAttachmentDefinition

func (net *netAttachDef) GetMetadata() *metav1.ObjectMeta {
	return &metav1.ObjectMeta{}
}

func init() {
	k8s.Register("k8s.cni.cncf.io", "v1", "network-attachment-definitions", true, &netAttachDef{})
}

func splitNamespaceNetwork(fullName string) (namespace, name string, err error) {
	s := strings.SplitN(fullName, "/", 2)
	switch len(s) {
	case 0:
		err = fmt.Errorf("no network specified")
		return
	case 1:
		namespace, err = getCurrentNamespace()
		name = fullName
		return
	case 2:
		namespace, name = s[0], s[1]
		return
	default:
		panic(fmt.Sprintf("impossible return value from strings.SplitN: %v", s))
	}
}

var (
	currentNamespaceOnce sync.Once
	currentNamespace     string
	currentNamespaceErr  error
)

func getCurrentNamespace() (string, error) {
	currentNamespaceOnce.Do(func() {
		var ns []byte
		ns, currentNamespaceErr = ioutil.ReadFile(
			"/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		currentNamespace = string(ns)
	})
	return currentNamespace, currentNamespaceErr
}
