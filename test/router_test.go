package aggregator_test

import (
	"testing"

	"traefik-config-middleware/pkg/aggregator"
)

func TestShouldIgnoreRouter_MatchingEntryPoint(t *testing.T) {
	router := aggregator.TraefikRouter{
		Name:        "test-router",
		EntryPoints: []string{"traefik"},
		Rule:        "PathPrefix(`/dashboard`)",
	}
	ignoreEntryPoints := []string{"traefik"}

	result := aggregator.ShouldIgnoreRouter(router, ignoreEntryPoints)

	if !result {
		t.Error("expected router to be ignored when entrypoint matches")
	}
}

func TestShouldIgnoreRouter_NoMatch(t *testing.T) {
	router := aggregator.TraefikRouter{
		Name:        "test-router",
		EntryPoints: []string{"websecure"},
		Rule:        "Host(`example.com`)",
	}
	ignoreEntryPoints := []string{"traefik", "internal"}

	result := aggregator.ShouldIgnoreRouter(router, ignoreEntryPoints)

	if result {
		t.Error("expected router NOT to be ignored when entrypoint doesn't match")
	}
}

func TestShouldIgnoreRouter_EmptyIgnoreList(t *testing.T) {
	router := aggregator.TraefikRouter{
		Name:        "test-router",
		EntryPoints: []string{"traefik"},
		Rule:        "PathPrefix(`/dashboard`)",
	}
	ignoreEntryPoints := []string{}

	result := aggregator.ShouldIgnoreRouter(router, ignoreEntryPoints)

	if result {
		t.Error("expected router NOT to be ignored when ignore list is empty")
	}
}

func TestShouldIgnoreRouter_NilIgnoreList(t *testing.T) {
	router := aggregator.TraefikRouter{
		Name:        "test-router",
		EntryPoints: []string{"traefik"},
		Rule:        "PathPrefix(`/dashboard`)",
	}

	result := aggregator.ShouldIgnoreRouter(router, nil)

	if result {
		t.Error("expected router NOT to be ignored when ignore list is nil")
	}
}

func TestShouldIgnoreRouter_MultipleEntryPoints_OneMatches(t *testing.T) {
	router := aggregator.TraefikRouter{
		Name:        "test-router",
		EntryPoints: []string{"websecure", "traefik"},
		Rule:        "Host(`example.com`)",
	}
	ignoreEntryPoints := []string{"traefik"}

	result := aggregator.ShouldIgnoreRouter(router, ignoreEntryPoints)

	if !result {
		t.Error("expected router to be ignored when any entrypoint matches")
	}
}

func TestShouldIgnoreRouter_MultipleEntryPoints_NoneMatch(t *testing.T) {
	router := aggregator.TraefikRouter{
		Name:        "test-router",
		EntryPoints: []string{"websecure", "web"},
		Rule:        "Host(`example.com`)",
	}
	ignoreEntryPoints := []string{"traefik", "internal"}

	result := aggregator.ShouldIgnoreRouter(router, ignoreEntryPoints)

	if result {
		t.Error("expected router NOT to be ignored when no entrypoints match")
	}
}

func TestShouldIgnoreRouter_MultipleIgnoreEntryPoints(t *testing.T) {
	router := aggregator.TraefikRouter{
		Name:        "test-router",
		EntryPoints: []string{"internal"},
		Rule:        "Host(`admin.example.com`)",
	}
	ignoreEntryPoints := []string{"traefik", "internal", "metrics"}

	result := aggregator.ShouldIgnoreRouter(router, ignoreEntryPoints)

	if !result {
		t.Error("expected router to be ignored when entrypoint is in ignore list")
	}
}

func TestShouldIgnoreRouter_EmptyRouterEntryPoints(t *testing.T) {
	router := aggregator.TraefikRouter{
		Name:        "test-router",
		EntryPoints: []string{},
		Rule:        "Host(`example.com`)",
	}
	ignoreEntryPoints := []string{"traefik"}

	result := aggregator.ShouldIgnoreRouter(router, ignoreEntryPoints)

	if result {
		t.Error("expected router NOT to be ignored when it has no entrypoints")
	}
}
