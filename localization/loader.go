package localization

import (
	"io/fs"
	"strings"
)

// FSLoader returns a ResourceLoader that reads FTL sources from an fs.FS (for
// example an embed.FS or a testing/fstest.MapFS).
//
// pathPattern is a template containing the placeholders "{locale}" and
// "{resource}", which are substituted with the requested locale and resource id
// to form the path passed to fs.ReadFile. For example:
//
//	FSLoader(fsys, "{locale}/{resource}.ftl")
//
// loads "de/main.ftl" for locale "de" and resource "main". Any error from
// fs.ReadFile (such as a missing file) is propagated to the caller, which
// treats it as a non-fatal, skipped resource.
func FSLoader(fsys fs.FS, pathPattern string) ResourceLoader {
	return func(locale, resourceID string) (string, error) {
		path := strings.NewReplacer("{locale}", locale, "{resource}", resourceID).Replace(pathPattern)
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}
