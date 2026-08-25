package sessionh

import (
	"context"
	"errors"
	"testing"

	session "github.com/matrixorigin/matrixflow/moi-backend/pkg/session"
	"github.com/stretchr/testify/require"
)

type semanticDependencyVolumesStub struct {
	roots map[int64]int64
	err   error
	seen  []int64
}

func (s *semanticDependencyVolumesStub) ResolveCanonicalRootVolume(_ context.Context, _ string, volumeID int64) (int64, error) {
	s.seen = append(s.seen, volumeID)
	if s.err != nil {
		return 0, s.err
	}
	return s.roots[volumeID], nil
}

func TestCoreSemanticModelDependencyResolverResolvesMixedSources(t *testing.T) {
	volumes := &semanticDependencyVolumesStub{roots: map[int64]int64{41: 9, 52: 9}}
	resolver := &CoreSemanticModelDependencyResolver{Volumes: volumes}
	targets, err := resolver.ResolveSourceDependencies(context.Background(), "ws-1", []session.CreateSemanticModelSourceRequest{
		{SourceType: semanticSourceLocalFile, FileName: "a.txt", FileID: "uploaded-file"},
		{SourceType: semanticSourceCatalogFile, FileID: "file-1", VolumeID: 41},
		{SourceType: semanticSourceCatalogFile, FileID: "file-2", VolumeID: 52},
		{SourceType: semanticSourceCatalogTable, TableID: 42},
		{SourceType: semanticSourceCatalogTable, TableID: 42},
	})
	require.NoError(t, err)
	require.Equal(t, []int64{41, 52}, volumes.seen)
	require.Len(t, targets, 2)
	require.Equal(t, "volume.read", targets[0].ActionID)
	require.Equal(t, "9", targets[0].Resource.ResourceID)
	require.Equal(t, "table.read", targets[1].ActionID)
	require.Equal(t, "42", targets[1].Resource.ResourceID)
}

func TestCoreSemanticModelDependencyResolverAuthorizesExplicitVolumeForMultiRootFile(t *testing.T) {
	// File linked under two roots: file-root containment would fail closed.
	// Supplying volume_id must authorize that root without ambiguity.
	volumes := &semanticDependencyVolumesStub{roots: map[int64]int64{41: 9001, 52: 9002}}
	resolver := &CoreSemanticModelDependencyResolver{Volumes: volumes}

	targets, err := resolver.ResolveSourceDependencies(context.Background(), "ws-1", []session.CreateSemanticModelSourceRequest{
		{SourceType: semanticSourceCatalogFile, FileID: "shared-file", VolumeID: 41},
	})
	require.NoError(t, err)
	require.Equal(t, []int64{41}, volumes.seen)
	require.Len(t, targets, 1)
	require.Equal(t, "volume.read", targets[0].ActionID)
	require.Equal(t, "9001", targets[0].Resource.ResourceID)

	// A second request for the same file under the other volume authorizes the other root.
	volumes.seen = nil
	targets, err = resolver.ResolveSourceDependencies(context.Background(), "ws-1", []session.CreateSemanticModelSourceRequest{
		{SourceType: semanticSourceCatalogFile, FileID: "shared-file", VolumeID: 52},
	})
	require.NoError(t, err)
	require.Equal(t, []int64{52}, volumes.seen)
	require.Equal(t, "9002", targets[0].Resource.ResourceID)
}

func TestCoreSemanticModelDependencyResolverDeduplicatesSharedRootVolumes(t *testing.T) {
	volumes := &semanticDependencyVolumesStub{roots: map[int64]int64{41: 9, 42: 9, 50: 10}}
	resolver := &CoreSemanticModelDependencyResolver{Volumes: volumes}
	targets, err := resolver.ResolveSourceDependencies(context.Background(), "ws-1", []session.CreateSemanticModelSourceRequest{
		{SourceType: semanticSourceCatalogFile, FileID: "file-a", VolumeID: 41},
		{SourceType: semanticSourceCatalogFile, FileID: "file-b", VolumeID: 42},
		{SourceType: semanticSourceCatalogFile, FileID: "file-c", VolumeID: 50},
	})
	require.NoError(t, err)
	require.Len(t, targets, 2)
	require.Equal(t, "9", targets[0].Resource.ResourceID)
	require.Equal(t, "10", targets[1].Resource.ResourceID)
}

func TestCoreSemanticModelDependencyResolverAcceptsUploadedLocalFileWithoutCatalogDependencies(t *testing.T) {
	targets, err := (&CoreSemanticModelDependencyResolver{}).ResolveSourceDependencies(context.Background(), "ws-1", []session.CreateSemanticModelSourceRequest{
		{SourceType: semanticSourceLocalFile, FileName: "a.txt", FileID: "uploaded-file"},
	})
	require.NoError(t, err)
	require.Empty(t, targets)
}

func TestCoreSemanticModelDependencyResolverResolvesSelectionParents(t *testing.T) {
	resolver := &CoreSemanticModelDependencyResolver{
		Volumes: &semanticDependencyVolumesStub{roots: map[int64]int64{11: 9}},
	}
	targets, err := resolver.ResolveSelectionDependencies(context.Background(), "ws-1", []session.SemanticModelSourceSelectionRequest{
		{Kind: semanticSelectionDatabase, DatabaseID: 42, AllSelected: true},
		{Kind: semanticSelectionVolume, VolumeID: 11, AllSelected: true},
		{Kind: semanticSelectionDatabase, DatabaseID: 42, SelectedTableIDs: []int64{7}},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"database.read", "volume.read"}, []string{targets[0].ActionID, targets[1].ActionID})
	require.Equal(t, "42", targets[0].Resource.ResourceID)
	require.Equal(t, "9", targets[1].Resource.ResourceID)
}

func TestCoreSemanticModelDependencyResolverSelectionFailsClosed(t *testing.T) {
	tests := []struct {
		name      string
		resolver  *CoreSemanticModelDependencyResolver
		selection session.SemanticModelSourceSelectionRequest
	}{
		{name: "unknown kind", resolver: &CoreSemanticModelDependencyResolver{}, selection: session.SemanticModelSourceSelectionRequest{Kind: "other", DatabaseID: 1}},
		{name: "database missing id", resolver: &CoreSemanticModelDependencyResolver{}, selection: session.SemanticModelSourceSelectionRequest{Kind: semanticSelectionDatabase}},
		{name: "volume missing resolver", resolver: &CoreSemanticModelDependencyResolver{}, selection: session.SemanticModelSourceSelectionRequest{Kind: semanticSelectionVolume, VolumeID: 1}},
		{name: "volume root unavailable", resolver: &CoreSemanticModelDependencyResolver{Volumes: &semanticDependencyVolumesStub{roots: map[int64]int64{}}}, selection: session.SemanticModelSourceSelectionRequest{Kind: semanticSelectionVolume, VolumeID: 1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.resolver.ResolveSelectionDependencies(context.Background(), "ws-1", []session.SemanticModelSourceSelectionRequest{tc.selection})
			require.Error(t, err)
		})
	}
}

func TestCoreSemanticModelDependencyResolverRejectsCatalogFileWithoutVolumeID(t *testing.T) {
	volumes := &semanticDependencyVolumesStub{roots: map[int64]int64{41: 9}}
	resolver := &CoreSemanticModelDependencyResolver{Volumes: volumes}
	_, err := resolver.ResolveSourceDependencies(context.Background(), "ws-1", []session.CreateSemanticModelSourceRequest{
		{SourceType: semanticSourceCatalogFile, FileID: "file-1"},
	})
	require.ErrorContains(t, err, "volume_id")
	require.Empty(t, volumes.seen)
}

func TestCoreSemanticModelDependencyResolverFailsClosedOnMissingVolumeResolver(t *testing.T) {
	resolver := &CoreSemanticModelDependencyResolver{}
	_, err := resolver.ResolveSourceDependencies(context.Background(), "ws-1", []session.CreateSemanticModelSourceRequest{
		{SourceType: semanticSourceCatalogFile, FileID: "file-1", VolumeID: 41},
	})
	require.ErrorContains(t, err, "volume resolver is unavailable")
}

func TestCoreSemanticModelDependencyResolverPropagatesCanonicalRootFailure(t *testing.T) {
	resolver := &CoreSemanticModelDependencyResolver{
		Volumes: &semanticDependencyVolumesStub{err: errors.New("cross workspace")},
	}
	_, err := resolver.ResolveSourceDependencies(context.Background(), "ws-1", []session.CreateSemanticModelSourceRequest{
		{SourceType: semanticSourceCatalogFile, FileID: "file-1", VolumeID: 41},
	})
	require.ErrorContains(t, err, "cross workspace")
}
