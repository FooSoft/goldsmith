# Goldsmith

Goldsmith is a fast and easily extensible static website generator written in Go. In contrast to many other generators, Goldsmith does not force any design paradigms or file organization rules on the user, making it possible to generate anything from blogs to image galleries using the same tool.

## Tutorial

Goldsmith does not use any configuration files, and all behavior customization happens in code. Goldsmith uses the [builder pattern](https://en.wikipedia.org/wiki/Builder_pattern) to establish a chain, which modifies files as they pass through it. Although the [Goldsmith](https://godoc.org/codeberg.org/foosoft/goldsmith) API is short and (hopefully) easy to understand, it is often best to learn by example:

1.  Start by copying files from a source directory to a destination directory (the simplest possible use case):

    ```go
    var gs goldsmith.Goldsmith
    gs.Begin(srcDir). // read files from srcDir
        End(dstDir)   // write files to dstDir
    ```

2.  Now let's convert any Markdown files to HTML fragments (while still copying the rest), using the [Markdown](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/markdown) plugin:

    ```go
    var gs goldsmith.Goldsmith
    gs.Begin(srcDir).          // read files from srcDir
        Chain(markdown.New()). // convert *.md files to *.html files
        End(dstDir)            // write files to dstDir
    ```

3.  If we have any [front matter](https://codeberg.org/foosoft/goldsmith-samples/raw/branch/master/basic/content/index.md) in our Markdown files, we need to extract it using the, [FrontMatter](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/frontmatter) plugin:

    ```go
    var gs goldsmith.Goldsmith
    gs.Begin(srcDir).             // read files from srcDir
        Chain(frontmatter.New()). // extract frontmatter and store it as metadata
        Chain(markdown.New()).    // convert *.md files to *.html files
        End(dstDir)               // write files to dstDir
    ```

4.  Next, we should run our barebones HTML through a [template](https://codeberg.org/foosoft/goldsmith-samples/raw/branch/master/basic/content/layouts/basic.gohtml) to add elements like a header, footer, or a menu; for this we can use the [Layout](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/frontmatter) plugin:

    ```go
    var gs goldsmith.Goldsmith
    gs.Begin(srcDir).             // read files from srcDir
        Chain(frontmatter.New()). // extract frontmatter and store it as metadata
        Chain(markdown.New()).    // convert *.md files to *.html files
        Chain(layout.New()).      // apply *.gohtml templates to *.html files
        End(dstDir)               // write files to dstDir
    ```

5.  Now, let's [minify](https://en.wikipedia.org/wiki/Minification_(programming)) our files to reduce data transfer and load times for our site's visitors using the [Minify](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/minify) plugin:

    ```go
    var gs goldsmith.Goldsmith
    gs.Begin(srcDir).             // read files from srcDir
        Chain(frontmatter.New()). // extract frontmatter and store it as metadata
        Chain(markdown.New()).    // convert *.md files to *.html files
        Chain(layout.New()).      // apply *.gohtml templates to *.html files
        Chain(minify.New()).      // minify *.html, *.css, *.js, etc. files
        End(dstDir)               // write files to dstDir
    ```

6.  Debugging problems in minified code can be tricky, so let's use the [Condition](https://godoc.org/codeberg.org/foosoft/goldsmith/filters/condition) filter to make minification occur only when we are ready for distribution.

    ```go
    var gs goldsmith.Goldsmith
    gs.Begin(srcDir).                    // read files from srcDir
        Chain(frontmatter.New()).        // extract frontmatter and store it as metadata
        Chain(markdown.New()).           // convert *.md files to *.html files
        Chain(layout.New()).             // apply *.gohtml templates to *.html files
        FilterPush(condition.New(dist)). // push a dist-only conditional filter onto the stack
        Chain(minify.New()).             // minify *.html, *.css, *.js, etc. files
        FilterPop().                     // pop off the last filter pushed onto the stack
        End(dstDir)                      // write files to dstDir
    ```

7.  Now that we have all of our plugins chained up, let's look at a complete example which uses [DevServer](https://godoc.org/codeberg.org/foosoft/goldsmith/devserver) to bootstrap a complete development sever which automatically rebuilds the site whenever source files are updated.

    ```go
    package main

    import (
        "flag"
        "log"

        "codeberg.org/foosoft/goldsmith"
        "codeberg.org/foosoft/goldsmith/devserver"
        "codeberg.org/foosoft/goldsmith/filters/condition"
        "codeberg.org/foosoft/goldsmith/plugins/frontmatter"
        "codeberg.org/foosoft/goldsmith/plugins/layout"
        "codeberg.org/foosoft/goldsmith/plugins/markdown"
        "codeberg.org/foosoft/goldsmith/plugins/minify"
    )

    type builder struct {
        dist bool
    }

    func (b *builder) Build(srcDir, dstDir, cacheDir string) {
        var gs goldsmith.Goldsmith
        errs := gs.Begin(srcDir).              // read files from srcDir
            Chain(frontmatter.New()).          // extract frontmatter and store it as metadata
            Chain(markdown.New()).             // convert *.md files to *.html files
            Chain(layout.New()).               // apply *.gohtml templates to *.html files
            FilterPush(condition.New(b.dist)). // push a dist-only conditional filter onto the stack
            Chain(minify.New()).               // minify *.html, *.css, *.js, etc. files
            FilterPop().                       // pop off the last filter pushed onto the stack
            End(dstDir)                        // write files to dstDir

        for _, err := range errs {
            log.Print(err)
        }
    }

    func main() {
        port := flag.Int("port", 8080, "server port")
        dist := flag.Bool("dist", false, "final dist mode")
        flag.Parse()

        devserver.DevServe(&builder{*dist}, *port, "content", "build", "cache")
    }
    ```

## Samples

Below are some examples of Goldsmith usage which can used to base your site on:

*   [Basic Sample](https://codeberg.org/foosoft/goldsmith-samples/src/branch/master/basic): a great starting point, this is the sample site from the tutorial.
*   [Bootstrap Sample](https://codeberg.org/foosoft/goldsmith-samples/src/branch/master/bootstrap): a slightly more advanced sample using [Bootstrap](https://getbootstrap.com/).
*   [FooSoft.net](https://codeberg.org/foosoft/goldsmith): I've been "dogfooding" Goldsmith by using it to [generate my homepage](/posts/generating-the-foosoft.net-homepage) for nearly a decade.

## Components

A growing set of plugins, filters, and other tools are provided to make it easier to get started with Goldsmith.

### Plugins

*   [Absolute](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/absolute): Convert relative HTML file references to absolute paths.
*   [Breadcrumbs](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/breadcrumbs): Generate metadata required to build breadcrumb navigation.
*   [Collection](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/collection): Group related pages into named collections.
*   [Document](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/document): Enable simple DOM modification via an API similar to jQuery.
*   [Forward](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/forward): Create simple redirections for pages that have moved to a new URL.
*   [FrontMatter](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/frontmatter): Extract the JSON, YAML, or TOML metadata stored in your files.
*   [Index](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/index): Create metadata for directory file listings and generate directory index pages.
*   [Layout](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/layout): Transform your HTML files with Go templates.
*   [LiveJs](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/livejs): Inject JavaScript code to automatically reload pages when modified.
*   [Markdown](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/markdown): Render Markdown documents as HTML fragments.
*   [Minify](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/minify): Remove superfluous data from a variety of web formats.
*   [Pager](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/pager): Split arrays of metadata into standalone pages.
*   [Rule](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/rule): Update metadata and filter files based on paths.
*   [Summary](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/summary): Extract summary and title metadata from HTML files.
*   [Syndicate](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/syndicate): Generate RSS, Atom, and JSON feeds from existing metadata.
*   [Syntax](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/syntax): Enable syntax highlighting for pre-formatted code blocks.
*   [Tags](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/tags): Generate tag clouds and indices from file metadata.
*   [Thumbnail](https://godoc.org/codeberg.org/foosoft/goldsmith/plugins/thumbnail): Build thumbnails for a variety of common image formats.

### Filters

*   [Condition](https://godoc.org/codeberg.org/foosoft/goldsmith/filters/condition): Filter files based on a single condition.
*   [Operator](https://godoc.org/codeberg.org/foosoft/goldsmith/filters/operator): Join filters using logical `AND`, `OR`, and `NOT` operators.
*   [Wildcard](https://godoc.org/codeberg.org/foosoft/goldsmith/filters/wildcard): Filter files using path wildcards (`*`, `?`, etc.)

### Other

*   [DevServer](https://godoc.org/codeberg.org/foosoft/goldsmith/devserver): Simple framework for building, updating, and viewing your site.
*   [Harness](https://godoc.org/codeberg.org/foosoft/goldsmith/harness): Unit test harness for verifying Goldsmith plugins and filters.
