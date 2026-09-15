package model

import "testing"

import "github.com/stretchr/testify/assert"

func TestRepositoryRefString(t *testing.T) {
	assert.Equal(t, "acme/project", (RepositoryRef{Owner: "acme", Name: "project"}).String())
}
