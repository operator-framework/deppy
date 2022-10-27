package factory

import (
	internalentitysource "github.com/operator-framework/deppy/internal/entitysource"
	pkgentitysource "github.com/operator-framework/deppy/pkg/entitysource"
)

func NewGroup(entitySources ...pkgentitysource.EntitySource) pkgentitysource.EntitySource {
	return internalentitysource.NewGroup(entitySources)
}

func NewCacheQuerier(entities map[pkgentitysource.EntityID]pkgentitysource.Entity) pkgentitysource.EntityQuerier {
	return internalentitysource.NewCacheQuerier(entities)
}
