package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	legacyrouter "github.com/getkin/kin-openapi/routers/legacy"
	"github.com/gofiber/fiber/v2"
)

func newOpenAPIMiddleware(specPath string) (fiber.Handler, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("load OpenAPI spec: %w", err)
	}

	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate OpenAPI spec: %w", err)
	}

	router, err := legacyrouter.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("create OpenAPI router: %w", err)
	}

	return func(c *fiber.Ctx) error {
		req, err := fiberToHTTPRequest(c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("openapi request adapt error: %v", err),
			})
		}

		route, pathParams, err := router.FindRoute(req)
		if err != nil {
			var routeErr *routers.RouteError
			if errors.As(err, &routeErr) {
				reason := routeErr.Error()
				if reason == routers.ErrPathNotFound.Error() || reason == routers.ErrMethodNotAllowed.Error() {
					return c.Next()
				}
			}

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("openapi routing error: %v", err),
			})
		}

		requestValidation := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
			Options: &openapi3filter.Options{
				AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
			},
		}

		if err := openapi3filter.ValidateRequest(c.Context(), requestValidation); err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Next()
	}, nil
}

func fiberToHTTPRequest(c *fiber.Ctx) (*http.Request, error) {
	bodyBytes := c.Body()

	req := &http.Request{
		Method:        c.Method(),
		Header:        fiberHeadersToHTTP(c),
		Body:          http.NoBody,
		ContentLength: int64(len(bodyBytes)),
	}

	if len(bodyBytes) > 0 {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	req.URL = &url.URL{
		Path:     c.Path(),
		RawQuery: string(c.Request().URI().QueryString()),
	}

	req.Host = c.Hostname()

	return req.WithContext(c.Context()), nil
}

func fiberHeadersToHTTP(c *fiber.Ctx) http.Header {
	headers := http.Header{}

	c.Request().Header.VisitAll(func(key, value []byte) {
		headers.Add(string(key), string(value))
	})

	return headers
}
