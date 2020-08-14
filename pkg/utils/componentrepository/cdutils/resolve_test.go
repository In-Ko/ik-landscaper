// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cdutils_test

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/pkg/utils/componentrepository/cdutils"
	mock_componentrepository "github.com/gardener/landscaper/pkg/utils/componentrepository/mock"
)

var _ = Describe("Resolve", func() {
	var (
		ctrl     *gomock.Controller
		cdClient *mock_componentrepository.MockClient

		repoCtx = []cdv2.RepositoryContext{
			{Type: cdv2.OCIRegistryType, BaseURL: "example.com"},
		}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		cdClient = mock_componentrepository.NewMockClient(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should contain the component itself in the component list", func() {
		ctx := context.Background()
		defer ctx.Done()
		cd := cdv2.ComponentDescriptor{}
		cd.RepositoryContexts = repoCtx
		cdClient.EXPECT().Resolve(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		list, err := cdutils.ResolveEffectiveComponentDescriptorList(ctx, cdClient, cd)
		Expect(err).ToNot(HaveOccurred())
		Expect(list.Components).To(ConsistOf(cd))
	})

	It("should resolve 2 direct transitive components", func() {
		ctx := context.Background()
		defer ctx.Done()

		l11_CD := cdv2.ComponentDescriptor{}
		l11_CD.RepositoryContexts = repoCtx
		l12_CD := cdv2.ComponentDescriptor{}
		l12_CD.RepositoryContexts = repoCtx

		cd := cdv2.ComponentDescriptor{}
		cd.RepositoryContexts = repoCtx
		cd.ComponentReferences = []cdv2.ObjectMeta{
			{
				Name:    "l11",
				Version: "0.0.1",
			},
			{
				Name:    "l12",
				Version: "0.0.1",
			},
		}

		cdClient.EXPECT().Resolve(ctx, repoCtx[0], cd.ComponentReferences[0]).Return(&l11_CD, nil)
		cdClient.EXPECT().Resolve(ctx, repoCtx[0], cd.ComponentReferences[1]).Return(&l12_CD, nil)

		list, err := cdutils.ResolveEffectiveComponentDescriptorList(ctx, cdClient, cd)
		Expect(err).ToNot(HaveOccurred())
		Expect(list.Components).To(ConsistOf(cd, l11_CD, l12_CD))
	})

	It("should recursively resolve transitive components", func() {
		ctx := context.Background()
		defer ctx.Done()

		l111_CD := cdv2.ComponentDescriptor{}
		l111_CD.RepositoryContexts = repoCtx

		l11_CD := cdv2.ComponentDescriptor{}
		l11_CD.RepositoryContexts = repoCtx
		l11_CD.ComponentReferences = []cdv2.ObjectMeta{
			{
				Name:    "l111",
				Version: "0.0.1",
			},
		}

		cd := cdv2.ComponentDescriptor{}
		cd.RepositoryContexts = repoCtx
		cd.ComponentReferences = []cdv2.ObjectMeta{
			{
				Name:    "l11",
				Version: "0.0.1",
			},
		}

		cdClient.EXPECT().Resolve(ctx, repoCtx[0], cd.ComponentReferences[0]).Return(&l11_CD, nil)
		cdClient.EXPECT().Resolve(ctx, repoCtx[0], l11_CD.ComponentReferences[0]).Return(&l111_CD, nil)

		list, err := cdutils.ResolveEffectiveComponentDescriptorList(ctx, cdClient, cd)
		Expect(err).ToNot(HaveOccurred())
		Expect(list.Components).To(ConsistOf(cd, l11_CD, l111_CD))
	})
})
