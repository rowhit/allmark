// Copyright 2013 Andreas Koch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package itemhandler

import (
	"fmt"
	"github.com/andreaskoch/allmark2/common/logger"
	"github.com/andreaskoch/allmark2/common/paths"
	"github.com/andreaskoch/allmark2/common/route"
	"github.com/andreaskoch/allmark2/model"
	"github.com/andreaskoch/allmark2/services/conversion"
	"github.com/andreaskoch/allmark2/ui/web/server/handler/handlerutil"
	"github.com/andreaskoch/allmark2/ui/web/server/index"
	"github.com/andreaskoch/allmark2/ui/web/view/templates"
	"github.com/andreaskoch/allmark2/ui/web/view/viewmodel"
	"io"
	"net/http"
)

func New(logger logger.Logger, index *index.Index, patherFactory paths.PatherFactory, converter conversion.Converter) *ItemHandler {
	return &ItemHandler{
		logger:        logger,
		index:         index,
		patherFactory: patherFactory,
		converter:     converter,
	}
}

type ItemHandler struct {
	logger        logger.Logger
	index         *index.Index
	patherFactory paths.PatherFactory
	converter     conversion.Converter
}

func (handler *ItemHandler) Func() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		// get the request route
		requestPath := handlerutil.GetRequestedPathFromRequest(r)
		requestRoute, err := route.NewFromRequest(requestPath)
		if err != nil {
			fmt.Fprintln(w, "%s", err)
			return
		}

		// make sure the request body is closed
		defer r.Body.Close()

		// check if there is a item for the request
		item, found := handler.index.IsMatch(*requestRoute)
		if !found {
			fmt.Fprintln(w, "item not found")
			return
		}

		// convert content
		pathProvider := handler.patherFactory.Relative()
		convertedContent, err := handler.converter.Convert(pathProvider, item)

		if err != nil {
			fmt.Fprintln(w, "Unable to convert content. Error: %s", err)
			return
		}

		// create a view model
		viewModel := viewmodel.Model{
			Type:                 item.Type.String(),
			Title:                item.Title,
			Description:          item.Description,
			Content:              convertedContent,
			ToplevelNavigation:   getToplevelNavigation(handler.index),
			BreadcrumbNavigation: getBreadcrumbNavigation(handler.index, item),
		}

		render(w, viewModel)

		// Childs
		childs := handler.index.GetChilds(item.Route())
		for _, child := range childs {
			fmt.Fprintf(w, "Child: %s\n", child.Title)
		}
	}
}

func getToplevelNavigation(index *index.Index) *viewmodel.ToplevelNavigation {
	root, err := route.NewFromRequest("")
	if err != nil {
		return nil
	}

	toplevelEntries := make([]*viewmodel.ToplevelEntry, 0)
	for _, child := range index.GetChilds(root) {

		// skip all childs which are not first level
		if child.Route().Level() != 1 {
			continue
		}

		toplevelEntries = append(toplevelEntries, &viewmodel.ToplevelEntry{
			Title: child.Title,
			Path:  child.Route().Value(),
		})

	}

	return &viewmodel.ToplevelNavigation{
		Entries: toplevelEntries,
	}
}

func getBreadcrumbNavigation(index *index.Index, item *model.Item) *viewmodel.BreadcrumbNavigation {

	// create a new bread crumb navigation
	navigation := &viewmodel.BreadcrumbNavigation{
		Entries: make([]*viewmodel.Breadcrumb, 0),
	}

	// abort if item or model is nil
	if item == nil {
		return navigation
	}

	// recurse if there is a parent
	if parent := index.GetParent(item.Route()); parent != nil {
		navigation.Entries = append(navigation.Entries, getBreadcrumbNavigation(index, parent).Entries...)
	}

	// append a new navigation entry and return it
	navigation.Entries = append(navigation.Entries, &viewmodel.Breadcrumb{
		Title: item.Title,
		Level: item.Route().Level(),
		Path:  item.Route().Value(),
	})

	return navigation
}

func render(writer io.Writer, viewModel viewmodel.Model) {

	templateProvider := templates.NewProvider(".")

	// get a template
	if template, err := templateProvider.GetFullTemplate(viewModel.Type); err == nil {

		err := template.Execute(writer, viewModel)
		if err != nil {
			fmt.Println(err)
		}

	} else {

		fmt.Fprintf(writer, "No template for item of type %q.", viewModel.Type)

	}

}