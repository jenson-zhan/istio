// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package webhooks

import (
	"bytes"
	"context"
	"strings"
	"testing"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMutatingWebhookPatch(t *testing.T) {
	ts := []struct {
		name        string
		configs     admissionregistrationv1beta1.MutatingWebhookConfigurationList
		configName  string
		webhookName string
		pemData     []byte
		err         string
	}{
		{
			"WebhookConfigNotFound",
			admissionregistrationv1beta1.MutatingWebhookConfigurationList{},
			"config1",
			"webhook1",
			[]byte("fake CA"),
			"\"config1\" not found",
		},
		{
			"WebhookEntryNotFound",
			admissionregistrationv1beta1.MutatingWebhookConfigurationList{
				Items: []admissionregistrationv1beta1.MutatingWebhookConfiguration{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "config1",
						},
					},
				},
			},
			"config1",
			"webhook1",
			[]byte("fake CA"),
			"webhook entry \"webhook1\" not found in config \"config1\"",
		},
		{
			"SuccessfullyPatched",
			admissionregistrationv1beta1.MutatingWebhookConfigurationList{
				Items: []admissionregistrationv1beta1.MutatingWebhookConfiguration{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "config1",
						},
						Webhooks: []admissionregistrationv1beta1.MutatingWebhook{
							{
								Name:         "webhook1",
								ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{},
							},
						},
					},
				},
			},
			"config1",
			"webhook1",
			[]byte("fake CA"),
			"",
		},
		{
			"MultipleWebhooks",
			admissionregistrationv1beta1.MutatingWebhookConfigurationList{
				Items: []admissionregistrationv1beta1.MutatingWebhookConfiguration{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "config1",
						},
						Webhooks: []admissionregistrationv1beta1.MutatingWebhook{
							{
								Name:         "webhook1",
								ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{},
							},
							{
								Name:         "webhook1",
								ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{},
							},
						},
					},
				},
			},
			"config1",
			"webhook1",
			[]byte("fake CA"),
			"",
		},
	}
	for _, tc := range ts {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewSimpleClientset(tc.configs.DeepCopyObject())
			err := patchMutatingWebhookConfig(client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations(),
				tc.configName, tc.webhookName, tc.pemData)
			if (err != nil) != (tc.err != "") {
				t.Fatalf("Wrong error: got %v want %v", err, tc.err)
			}
			if err != nil {
				if !strings.Contains(err.Error(), tc.err) {
					t.Fatalf("Got %q, want %q", err, tc.err)
				}
			} else {
				obj, err := client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(context.Background(), tc.configName, metav1.GetOptions{})
				if err != nil {
					t.Fatal(err)
				}
				for _, w := range obj.Webhooks {
					if !bytes.Equal(w.ClientConfig.CABundle, tc.pemData) {
						t.Fatalf("Incorrect CA bundle: expect %s got %s", tc.pemData, w.ClientConfig.CABundle)
					}
				}
			}
		})
	}
}
