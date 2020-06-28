package changelog

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestChangelogProvidedViaFlag(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.ReleaseNotes = "testdata/changes.md"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "c0ff33 coffeee\n", ctx.ReleaseNotes)
}

func TestTemplatedChangelogProvidedViaFlag(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.ReleaseNotes = "testdata/changes-templated.md"
	ctx.Git.CurrentTag = "v0.0.1"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "c0ff33 coffeee v0.0.1\n", ctx.ReleaseNotes)
}

func TestChangelogProvidedViaFlagAndSkipEnabled(t *testing.T) {
	var ctx = context.New(config.Project{
		Changelog: config.Changelog{
			Skip: true,
		},
	})
	ctx.ReleaseNotes = "testdata/changes.md"
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, "c0ff33 coffeee\n", ctx.ReleaseNotes)
}

func TestChangelogProvidedViaFlagDoesntExist(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.ReleaseNotes = "testdata/changes.nope"
	err := Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "open testdata/changes.nope: ")
}

func TestChangelogSkip(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.Config.Changelog.Skip = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestReleaseHeaderProvidedViaFlagDoesntExist(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.ReleaseHeader = "testdata/header.nope"
	err := Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "open testdata/header.nope: ")
}

func TestReleaseFooterProvidedViaFlagDoesntExist(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.ReleaseFooter = "testdata/footer.nope"
	err := Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "open testdata/footer.nope: ")
}

func TestSnapshot(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.Snapshot = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestChangelog(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "added feature 1")
	testlib.GitCommit(t, "fixed bug 2")
	testlib.GitCommit(t, "ignored: whatever")
	testlib.GitCommit(t, "docs: whatever")
	testlib.GitCommit(t, "something about cArs we dont need")
	testlib.GitCommit(t, "feat: added that thing")
	testlib.GitCommit(t, "Merge pull request #999 from goreleaser/some-branch")
	testlib.GitCommit(t, "this is not a Merge pull request")
	testlib.GitTag(t, "v0.0.2")
	var ctx = context.New(config.Project{
		Dist: folder,
		Changelog: config.Changelog{
			Filters: config.Filters{
				Exclude: []string{
					"docs:",
					"ignored:",
					"(?i)cars",
					"^Merge pull request",
				},
			},
		},
	})
	ctx.Git.CurrentTag = "v0.0.2"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.NotContains(t, ctx.ReleaseNotes, "first")
	require.Contains(t, ctx.ReleaseNotes, "added feature 1")
	require.Contains(t, ctx.ReleaseNotes, "fixed bug 2")
	require.NotContains(t, ctx.ReleaseNotes, "docs")
	require.NotContains(t, ctx.ReleaseNotes, "ignored")
	require.NotContains(t, ctx.ReleaseNotes, "cArs")
	require.NotContains(t, ctx.ReleaseNotes, "from goreleaser/some-branch")

	bts, err := ioutil.ReadFile(filepath.Join(folder, "CHANGELOG.md"))
	require.NoError(t, err)
	require.NotEmpty(t, string(bts))
}

func TestChangelogPreviousTagEnv(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "second")
	testlib.GitTag(t, "v0.0.2")
	testlib.GitCommit(t, "third")
	testlib.GitTag(t, "v0.0.3")
	var ctx = context.New(config.Project{
		Dist:      folder,
		Changelog: config.Changelog{Filters: config.Filters{}},
	})
	ctx.Git.CurrentTag = "v0.0.3"
	require.NoError(t, os.Setenv("GORELEASER_PREVIOUS_TAG", "v0.0.1"))
	require.NoError(t, Pipe{}.Run(ctx))
	require.NoError(t, os.Setenv("GORELEASER_PREVIOUS_TAG", ""))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.NotContains(t, ctx.ReleaseNotes, "first")
	require.Contains(t, ctx.ReleaseNotes, "second")
	require.Contains(t, ctx.ReleaseNotes, "third")
}

func TestChangelogForGitlab(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "added feature 1")
	testlib.GitCommit(t, "fixed bug 2")
	testlib.GitCommit(t, "ignored: whatever")
	testlib.GitCommit(t, "docs: whatever")
	testlib.GitCommit(t, "something about cArs we dont need")
	testlib.GitCommit(t, "feat: added that thing")
	testlib.GitCommit(t, "Merge pull request #999 from goreleaser/some-branch")
	testlib.GitCommit(t, "this is not a Merge pull request")
	testlib.GitTag(t, "v0.0.2")
	var ctx = context.New(config.Project{
		Dist: folder,
		Changelog: config.Changelog{
			Filters: config.Filters{
				Exclude: []string{
					"docs:",
					"ignored:",
					"(?i)cars",
					"^Merge pull request",
				},
			},
		},
	})
	ctx.TokenType = context.TokenTypeGitLab
	ctx.Git.CurrentTag = "v0.0.2"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.NotContains(t, ctx.ReleaseNotes, "first")
	require.Contains(t, ctx.ReleaseNotes, "added feature 1") // no whitespace because its the last entry of the changelog
	require.Contains(t, ctx.ReleaseNotes, "fixed bug 2   ")  // whitespaces are on purpose
	require.NotContains(t, ctx.ReleaseNotes, "docs")
	require.NotContains(t, ctx.ReleaseNotes, "ignored")
	require.NotContains(t, ctx.ReleaseNotes, "cArs")
	require.NotContains(t, ctx.ReleaseNotes, "from goreleaser/some-branch")

	bts, err := ioutil.ReadFile(filepath.Join(folder, "CHANGELOG.md"))
	require.NoError(t, err)
	require.NotEmpty(t, string(bts))
}

func TestChangelogSort(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "whatever")
	testlib.GitTag(t, "v0.9.9")
	testlib.GitCommit(t, "c: commit")
	testlib.GitCommit(t, "a: commit")
	testlib.GitCommit(t, "b: commit")
	testlib.GitTag(t, "v1.0.0")
	var ctx = context.New(config.Project{
		Changelog: config.Changelog{},
	})
	ctx.Git.CurrentTag = "v1.0.0"

	for _, cfg := range []struct {
		Sort    string
		Entries []string
	}{
		{
			Sort: "",
			Entries: []string{
				"b: commit",
				"a: commit",
				"c: commit",
			},
		},
		{
			Sort: "asc",
			Entries: []string{
				"a: commit",
				"b: commit",
				"c: commit",
			},
		},
		{
			Sort: "desc",
			Entries: []string{
				"c: commit",
				"b: commit",
				"a: commit",
			},
		},
	} {
		t.Run("changelog sort='"+cfg.Sort+"'", func(t *testing.T) {
			ctx.Config.Changelog.Sort = cfg.Sort
			entries, err := buildChangelog(ctx)
			require.NoError(t, err)
			require.Len(t, entries, len(cfg.Entries))
			var changes []string
			for _, line := range entries {
				changes = append(changes, extractCommitInfo(line))
			}
			require.EqualValues(t, cfg.Entries, changes)
		})
	}
}

func TestChangelogInvalidSort(t *testing.T) {
	var ctx = context.New(config.Project{
		Changelog: config.Changelog{
			Sort: "dope",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), ErrInvalidSortDirection.Error())
}

func TestChangelogOfFirstRelease(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	var msgs = []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	var ctx = context.New(config.Project{})
	ctx.Git.CurrentTag = "v0.0.1"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	for _, msg := range msgs {
		require.Contains(t, ctx.ReleaseNotes, msg)
	}
}

func TestChangelogFilterInvalidRegex(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commitssss")
	testlib.GitTag(t, "v0.0.3")
	testlib.GitCommit(t, "commitzzz")
	testlib.GitTag(t, "v0.0.4")
	var ctx = context.New(config.Project{
		Changelog: config.Changelog{
			Filters: config.Filters{
				Exclude: []string{
					"(?iasdr4qasd)not a valid regex i guess",
				},
			},
		},
	})
	ctx.Git.CurrentTag = "v0.0.4"
	require.EqualError(t, Pipe{}.Run(ctx), "error parsing regexp: invalid or unsupported Perl syntax: `(?ia`")
}

func TestChangelogNoTags(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	var ctx = context.New(config.Project{})
	require.Error(t, Pipe{}.Run(ctx))
	require.Empty(t, ctx.ReleaseNotes)
}

func TestChangelogOnBranchWithSameNameAsTag(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	var msgs = []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCheckoutBranch(t, "v0.0.1")
	var ctx = context.New(config.Project{})
	ctx.Git.CurrentTag = "v0.0.1"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	for _, msg := range msgs {
		require.Contains(t, ctx.ReleaseNotes, msg)
	}
}

func CopyDirectory(source, target string) error {
	if runtime.GOOS == "windows" {
		args := []string{"xcopy", "/I", "/E", source, target}
		b, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%q: %w:\n%s", args, err, string(b))
		}
	}
	_, err := exec.Command("cp", "-Rf", source, target).CombinedOutput()
	return err
}

func TestChangeLogWithReleaseHeader(t *testing.T) {
	current, err := os.Getwd()
	require.NoError(t, err)
	tmpdir, back := testlib.Mktmp(t)
	defer back()

	require.NoError(t, CopyDirectory(filepath.Join(current, "testdata"), filepath.Join(tmpdir, "testdata")))
	testlib.GitInit(t)
	var msgs = []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCheckoutBranch(t, "v0.0.1")
	var ctx = context.New(config.Project{})
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.ReleaseHeader = "testdata/release-header.md"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.Contains(t, ctx.ReleaseNotes, "test header")
}

func TestChangeLogWithTemplatedReleaseHeader(t *testing.T) {
	current, err := os.Getwd()
	require.NoError(t, err)
	tmpdir, back := testlib.Mktmp(t)
	defer back()
	require.NoError(t, CopyDirectory(filepath.Join(current, "testdata"), filepath.Join(tmpdir, "testdata")))
	testlib.GitInit(t)
	var msgs = []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCheckoutBranch(t, "v0.0.1")
	var ctx = context.New(config.Project{})
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.ReleaseHeader = "testdata/release-header-templated.md"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.Contains(t, ctx.ReleaseNotes, "test header with tag v0.0.1")
}
func TestChangeLogWithReleaseFooter(t *testing.T) {
	current, err := os.Getwd()
	require.NoError(t, err)
	tmpdir, back := testlib.Mktmp(t)
	defer back()
	require.NoError(t, CopyDirectory(filepath.Join(current, "testdata"), filepath.Join(tmpdir, "testdata")))
	testlib.GitInit(t)
	var msgs = []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCheckoutBranch(t, "v0.0.1")
	var ctx = context.New(config.Project{})
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.ReleaseFooter = "testdata/release-footer.md"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.Contains(t, ctx.ReleaseNotes, "test footer")
}

func TestChangeLogWithTemplatedReleaseFooter(t *testing.T) {
	current, err := os.Getwd()
	require.NoError(t, err)
	tmpdir, back := testlib.Mktmp(t)
	defer back()
	require.NoError(t, CopyDirectory(filepath.Join(current, "testdata"), filepath.Join(tmpdir, "testdata")))
	testlib.GitInit(t)
	var msgs = []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCheckoutBranch(t, "v0.0.1")
	var ctx = context.New(config.Project{})
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.ReleaseFooter = "testdata/release-footer-templated.md"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.Contains(t, ctx.ReleaseNotes, "test footer with tag v0.0.1")
}
