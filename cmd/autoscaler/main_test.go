/*
Copyright 2018 The Knative Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/knative/serving/pkg/apis/serving"
	"github.com/knative/serving/pkg/autoscaler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	fakeK8s "k8s.io/client-go/kubernetes/fake"
)

const (
	testNamespace = "test-namespace"
	testRevision  = "test-Revision"
)

func TestUniscalerFactoryFailures(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   string
	}{{
		"nil labels", nil, fmt.Sprintf("label %q not found or empty in Decider", serving.RevisionLabelKey),
	}, {
		"empty labels", map[string]string{}, fmt.Sprintf("label %q not found or empty in Decider", serving.RevisionLabelKey),
	}, {
		"revision missing", map[string]string{
			serving.ServiceLabelKey:       "in vino",
			serving.ConfigurationLabelKey: "veritas",
		},
		fmt.Sprintf("label %q not found or empty in Decider", serving.RevisionLabelKey),
	}, {
		"config missing", map[string]string{
			serving.RevisionLabelKey: "en el vino",
			serving.ServiceLabelKey:  "está la verdad",
		},
		fmt.Sprintf("label %q not found or empty in Decider", serving.ConfigurationLabelKey),
	}, {
		"values not ascii", map[string]string{
			serving.RevisionLabelKey:      "dans le vin",
			serving.ServiceLabelKey:       "la",
			serving.ConfigurationLabelKey: "verité",
		}, "invalid value: only ASCII characters accepted",
	}, {
		"too long of a value", map[string]string{
			serving.RevisionLabelKey:      "the",
			serving.ServiceLabelKey:       "cat is ",
			serving.ConfigurationLabelKey: "l" + strings.Repeat("o", 253) + "ng",
		}, "max length must be 255 characters",
	}}

	uniScalerFactory := getTestUniScalerFactory()
	decider := &autoscaler.Decider{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testRevision,
		},
	}
	dynamicConfig := autoscaler.NewDynamicConfig(&autoscaler.Config{
		KeepAliveTimes: 2,
	}, zap.NewNop().Sugar())

	for _, test := range tests {
		decider.Labels = test.labels

		_, err := uniScalerFactory(decider, dynamicConfig)
		if err == nil {
			t.Fatal("No error was returned")
		}
		if got, want := err.Error(), test.want; !strings.Contains(got, want) {
			t.Errorf("Error = %q, want to contain = %q", got, want)
		}
	}
}

func TestUniScalerFactoryFunc(t *testing.T) {
	uniScalerFactory := getTestUniScalerFactory()
	for _, srv := range []string{"some", ""} {
		decider := &autoscaler.Decider{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testRevision,
				Labels: map[string]string{
					serving.RevisionLabelKey:      testRevision,
					serving.ServiceLabelKey:       srv,
					serving.ConfigurationLabelKey: "test-config",
				},
			},
		}
		dynamicConfig := autoscaler.NewDynamicConfig(&autoscaler.Config{
			KeepAliveTimes: 2,
		}, zap.NewNop().Sugar())

		if _, err := uniScalerFactory(decider, dynamicConfig); err != nil {
			t.Errorf("got error from uniScalerFactory: %v", err)
		}
	}
}

func getTestUniScalerFactory() func(decider *autoscaler.Decider, dynamicConfig *autoscaler.DynamicConfig) (autoscaler.UniScaler, error) {
	kubeClient := fakeK8s.NewSimpleClientset()
	kubeInformer := kubeinformers.NewSharedInformerFactory(kubeClient, 0)
	return uniScalerFactoryFunc(kubeInformer.Core().V1().Endpoints())
}
