// Copyright 2019 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hugolib

import (
	"github.com/yezige/hugo/output"
	"github.com/yezige/hugo/resources/page"
	"github.com/yezige/hugo/resources/resource"
)

func newPageOutput(
	ps *pageState,
	pp pagePaths,
	f output.Format,
	render bool) *pageOutput {
	var targetPathsProvider targetPathsHolder
	var linksProvider resource.ResourceLinksProvider

	ft, found := pp.targetPaths[f.Name]
	if !found {
		// Link to the main output format
		ft = pp.targetPaths[pp.firstOutputFormat.Format.Name]
	}
	targetPathsProvider = ft
	linksProvider = ft

	var paginatorProvider page.PaginatorProvider = page.NopPage
	var pag *pagePaginator

	if render && ps.IsNode() {
		pag = newPagePaginator(ps)
		paginatorProvider = pag
	}

	providers := struct {
		page.PaginatorProvider
		resource.ResourceLinksProvider
		targetPather
	}{
		paginatorProvider,
		linksProvider,
		targetPathsProvider,
	}

	po := &pageOutput{
		f:                       f,
		pagePerOutputProviders:  providers,
		ContentProvider:         page.NopPage,
		PageRenderProvider:      page.NopPage,
		TableOfContentsProvider: page.NopPage,
		render:                  render,
		paginator:               pag,
	}

	return po
}

// We create a pageOutput for every output format combination, even if this
// particular page isn't configured to be rendered to that format.
type pageOutput struct {
	// Set if this page isn't configured to be rendered to this format.
	render bool

	f output.Format

	// Only set if render is set.
	// Note that this will be lazily initialized, so only used if actually
	// used in template(s).
	paginator *pagePaginator

	// These interface provides the functionality that is specific for this
	// output format.
	contentRenderer page.ContentRenderer
	pagePerOutputProviders
	page.ContentProvider
	page.PageRenderProvider
	page.TableOfContentsProvider
	page.RenderShortcodesProvider

	// May be nil.
	cp *pageContentOutput
}

func (p *pageOutput) initContentProvider(cp *pageContentOutput) {
	if cp == nil {
		return
	}
	p.contentRenderer = cp
	p.ContentProvider = cp
	p.PageRenderProvider = cp
	p.TableOfContentsProvider = cp
	p.RenderShortcodesProvider = cp
	p.cp = cp

}

func (p *pageOutput) enablePlaceholders() {
	if p.cp != nil {
		p.cp.enablePlaceholders()
	}
}
