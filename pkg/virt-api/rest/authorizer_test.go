/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2018 Red Hat, Inc.
 *
 */

package rest

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/url"

	"github.com/emicklei/go-restful"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
)

var _ = Describe("Authorizer", func() {

	Describe("VirtualMachineInstance Subresources", func() {
		var (
			req        *restful.Request
			sarHandler func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error)
			app        VirtApiAuthorizor
		)

		BeforeEach(func() {
			req = &restful.Request{}
			req.Request = &http.Request{}
			req.Request.URL = &url.URL{}
			req.Request.Header = make(map[string][]string)
			req.Request.Header[userHeader] = []string{"user"}
			req.Request.Header[groupHeader] = []string{"userGroup"}
			req.Request.Header[userExtraHeaderPrefix+"test"] = []string{"userExtraValue"}
			req.Request.Method = http.MethodGet
			req.Request.URL.Path = "/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi/console"
			req.Request.TLS = &tls.ConnectionState{}
			req.Request.TLS.PeerCertificates = append(req.Request.TLS.PeerCertificates, &x509.Certificate{})

			sarHandler = func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
				panic("unexpected call to sarHandler")
			}

			kubeClient := fake.NewSimpleClientset()
			kubeClient.Fake.PrependReactor("create", "subjectaccessreviews", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				sar, ok := create.GetObject().(*authv1.SubjectAccessReview)
				Expect(ok).To(BeTrue())

				sarOut, err := sarHandler(sar)
				return true, sarOut, err
			})

			app = NewAuthorizorFromClient(kubeClient.AuthorizationV1().SubjectAccessReviews())
		})

		Context("Subresource api", func() {
			It("should reject unauthenticated user", func() {
				req.Request.TLS = nil

				allowed, reason, err := app.Authorize(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(allowed).To(BeFalse())
				Expect(reason).To(Equal("request is not authenticated"))
			})

			It("should reject unauthorized user", func() {
				sarHandler = func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
					sar.Status.Allowed = false
					sar.Status.Reason = "just because"
					return sar, nil
				}

				allowed, reason, err := app.Authorize(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(allowed).To(BeFalse())
				Expect(reason).To(Equal("just because"))
			})

			It("should allow authorized user", func() {
				sarHandler = func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
					sar.Status.Allowed = true
					return sar, nil
				}

				allowed, _, err := app.Authorize(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(allowed).To(BeTrue())
			})

			It("should not allow user if auth check fails", func() {
				sarHandler = func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
					return nil, errors.New("internal error")
				}

				allowed, _, err := app.Authorize(req)
				Expect(err).To(HaveOccurred())
				Expect(allowed).To(BeFalse())
			})

			Context("with non-resource endpoint", func() {
				const endpointPath = "/apis/subresources.kubevirt.io/v1alpha3/expand-spec"

				It("should allow authorized user", func() {
					req.Request.Method = http.MethodGet
					req.Request.URL.Path = endpointPath

					sarHandler = func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
						Expect(sar.Spec.NonResourceAttributes).ToNot(BeNil())
						Expect(sar.Spec.NonResourceAttributes.Verb).To(Equal(http.MethodGet))
						Expect(sar.Spec.NonResourceAttributes.Path).To(Equal(endpointPath))

						sar.Status.Allowed = true
						return sar, nil
					}

					allowed, _, err := app.Authorize(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(allowed).To(BeTrue())
				})

				It("should not allow unauthorized user", func() {
					req.Request.Method = http.MethodGet
					req.Request.URL.Path = endpointPath

					sarHandler = func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
						Expect(sar.Spec.NonResourceAttributes).ToNot(BeNil())
						Expect(sar.Spec.NonResourceAttributes.Verb).To(Equal(http.MethodGet))
						Expect(sar.Spec.NonResourceAttributes.Path).To(Equal(endpointPath))

						sar.Status.Allowed = false
						sar.Status.Reason = "unauthorized"
						return sar, nil
					}

					allowed, reason, err := app.Authorize(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(allowed).To(BeFalse())
					Expect(reason).To(Equal("unauthorized"))
				})
			})

			DescribeTable("should allow all users for info endpoints", func(path string) {
				req.Request.TLS = nil
				req.Request.URL.Path = path
				allowed, _, err := app.Authorize(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(allowed).To(BeTrue())
			},
				Entry("root", "/"),
				Entry("apis", "/apis"),
				Entry("group", "/apis/subresources.kubevirt.io"),
				Entry("version", "/apis/subresources.kubevirt.io/version"),
				Entry("healthz", "/apis/subresources.kubevirt.io/healthz"),
				Entry("start profiler", "/apis/subresources.kubevirt.io/start-cluster-profiler"),
				Entry("stop profiler", "/apis/subresources.kubevirt.io/stop-cluster-profiler"),
				Entry("dump profiler", "/apis/subresources.kubevirt.io/dump-cluster-profiler"),
			)

			DescribeTable("should reject all users for unknown endpoint paths", func(path string) {
				req.Request.URL.Path = path
				allowed, _, err := app.Authorize(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(allowed).To(BeFalse())
			},
				Entry("too short resource specific endpoint", "/apis/subresources.kubevirt.io/v1alpha3/namespaces/madethisup"),
				Entry("no subresource provided", "/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi"),
				Entry("invalid resource type", "/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/madeupresource/testvmi/console"),
			)
		})
	})

	DescribeTable("should map verbs", func(httpVerb string, resourceName string, expectedRbacVerb string) {
		Expect(mapHttpVerbToRbacVerb(httpVerb, resourceName)).To(Equal(expectedRbacVerb))
	},
		Entry("http post to create", http.MethodPost, "", "create"),
		Entry("http get with resource to get", http.MethodGet, "foo", "get"),
		Entry("http get without resource to list", http.MethodGet, "", "list"),
		Entry("http put to update", http.MethodPut, "", "update"),
		Entry("http patch to patch", http.MethodPatch, "", "patch"),
		Entry("http delete with reource to delete", http.MethodDelete, "foo", "delete"),
		Entry("http delete without resource to deletecollection", http.MethodDelete, "", "deletecollection"),
	)

})
