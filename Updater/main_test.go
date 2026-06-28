package main

import (
	"reflect"
	"testing"
)

func TestUpdaterComposeArgsUsesStableDefaultProject(t *testing.T) {
	t.Setenv("UPDATER_PROJECT_NAME", "")
	t.Setenv("UPDATER_COMPOSE_FILES", "")

	want := []string{"compose", "--project-name", "wavenode", "-f", "/compose/docker-compose.yml"}
	if got := updaterComposeArgs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("updaterComposeArgs() = %#v, want %#v", got, want)
	}
}

func TestUpdaterComposeArgsUsesConfiguredProjectAndFiles(t *testing.T) {
	t.Setenv("UPDATER_PROJECT_NAME", "music-stack")
	t.Setenv("UPDATER_COMPOSE_FILES", "/compose/docker-compose.yml, /compose/docker-compose.internet.yml")

	want := []string{
		"compose", "--project-name", "music-stack",
		"-f", "/compose/docker-compose.yml",
		"-f", "/compose/docker-compose.internet.yml",
	}
	if got := updaterComposeArgs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("updaterComposeArgs() = %#v, want %#v", got, want)
	}
}
