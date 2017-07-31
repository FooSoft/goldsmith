# Goldsmith #

Goldsmith is a static website generator developed in Go with flexibility, extensibility, and performance as primary
design considerations. With Goldsmith you can easily build and deploy any type of site, whether it is a personal blog,
image gallery, or a corporate homepage; the tool no assumptions are made about your layout or file structure. Goldsmith
is trivially extensible via a plugin architecture which makes it simple to perform complex data transformations
concurrently.

Naturally, I use Goldsmith to generate my personal website, [FooSoft Productions](https://foosoft.net/). If you would
like to know how a Bootstrap site can be put together with this static generator, you can check out the [source content
files](https://github.com/FooSoft/foosoft.net.git) as well as the [plugin
chain](https://github.com/FooSoft/webtools/blob/master/webbuild/main.go) that makes everything happen.

![](https://foosoft.net/projects/goldsmith/img/gold.png)

## Motivation ##

Why in the world would one create yet another static site generator? At first, I didn't think I needed to; after all,
there is a wide variety of open source tools freely available for use. Surely one of these applications would allow me
to build my portfolio page exactly the way I want right?

After trying several static generators, namely [Pelican](http://blog.getpelican.com/), [Hexo](https://hexo.io/), and
[Hugo](https://gohugo.io/), I found that although sometimes coming close, no tool gave me exactly what I needed.
Although I hold the authors of these applications in high regard and sincerely appreciate their contribution to the open
source community, everyone seemed overly eager to make assumptions about content organization and presentation.

Many of the static generators I've used feature extensive configuration files to support customization. Nevertheless, I
was disappointed to discovered that even though I could approach my planned design, I could never realize it. There was
always some architectural limitation preventing me from doing something with my site which seemed like basic
functionality.

*   Blog posts can be tagged, but static pages cannot.
*   Image files cannot be stored next to content files.
*   Navbar item activated when viewing blog, but not static pages.
*   Auto-generated pages behave differently from normal ones.

Upon asking on community forms, I learned that most users were content to live with such design decisions, with some
offering workarounds that would get me halfway to where I wanted to go. As I am not one to make compromises, I kept
hopping from one static site generator to another, until I discovered [Metalsmith](http://www.metalsmith.io/). Finally,
it seemed like I found a tool that gets out of my way, and lets me build my website the way I want to. After using this
tool for almost a year, I began to see its limits.

*   The extension system is complicated; it's difficult to write and debug plugins.
*   Quality of existing plugins varies greatly; I found many subtle issues.
*   No support for parallel processing (this is a big one if you process images).
*   A full Node.js is stack (including dependencies) is required to build sites.

Rather than making do with what I had indefinitely, I decided to use the knowledge I've obtained from using various
static site generators to build my own. The *Goldsmith* name is a reference to both the *Go* programming language I've
selected for this project, as well as to *Metalsmith*, my inspiration for what an static site generator could be.

The motivation behind Goldsmith can be described by the following principles:

*   Keep the core small and simple.
*   Enable efficient, multi-core processing.
*   Add new features via user plugins.
*   Customize behavior through user code.

I originally built this tool to generate my personal homepage, but I believe it can be of use to anyone who wants to
enjoy the freedom of building a static site from ground up, especially users of Metalsmith. Why craft metal when you can
be crafting gold?

## Plugins ##

A growing set of core plugins is provided to make it easier to get started with this tool to generate static websites.

*   **[Goldsmith-Abs](https://foosoft.net/projects/goldsmith/plugins/abs/)**: Convert HTML relative file references to absolute paths.
*   **[Goldsmith-Breadcrumbs](https://foosoft.net/projects/goldsmith/plugins/breadcrumbs/)**: Manage metadata required to build navigation breadcrumbs.
*   **[Goldsmith-Collection](https://foosoft.net/projects/goldsmith/plugins/collection/)**: Group related pages into named collections.
*   **[Goldsmith-Condition](https://foosoft.net/projects/goldsmith/plugins/condition/)**: Conditionally chain plugins based on various criteria.
*   **[Goldsmith-FrontMatter](https://foosoft.net/projects/goldsmith/plugins/frontmatter/)**: Extract front matter from files and store it in file metadata.
*   **[Goldsmith-Include](https://foosoft.net/projects/goldsmith/plugins/include/)**: Include additional paths for processing.
*   **[Goldsmith-Index](https://foosoft.net/projects/goldsmith/plugins/index/)**: Create index pages for displaying directory listings.
*   **[Goldsmith-Layout](https://foosoft.net/projects/goldsmith/plugins/layout/)**: Process partial HTML into complete pages with Go templates.
*   **[Goldsmith-LiveJs](https://foosoft.net/projects/goldsmith/plugins/livejs/)**: Automatically refresh your web browser page on content change.
*   **[Goldsmith-Markdown](https://foosoft.net/projects/goldsmith/plugins/markdown/)**: Process Markdown files to generate partial HTML documents.
*   **[Goldsmith-Minify](https://foosoft.net/projects/goldsmith/plugins/minify/)**: Reduce the data size of various web file formats.
*   **[Goldsmith-Tags](https://foosoft.net/projects/goldsmith/plugins/tags/)**: Generate metadata and index pages for tags.
*   **[Goldsmith-Thumbnail](https://foosoft.net/projects/goldsmith/plugins/thumbnail/)**: Generate thumbnails for a variety of image formats.

## Usage ##

Goldsmith is a pipeline-based file processor. Files are loaded in from the source directory, processed by a number of
plugins, and are finally output to the destination directory. Rather than explaining the process in detail conceptually,
I will show some code samples which show how this tool can be used in practice.

*   Start by copying files from a source directory to a destination directory (simplest possible use case):

    ```
    goldsmith.Begin(srcDir).
        End(dstDir)
    ```

*   Now let's also convert our Markdown files to HTML using the
    [Goldsmith-Markdown](https://foosoft.net/projects/goldsmith/plugins/markdown/) plugin:

    ```
    goldsmith.Begin(srcDir).
        Chain(markdown.NewCommon()).
        End(dstDir)
    ```

*   If we are using *frontmatter* in our Markdown files, we can easily extract it by using the
    [Goldsmith-Frontmatter](https://foosoft.net/projects/goldsmith/plugins/frontmatter/) plugin:

    ```
    goldsmith.Begin(srcDir).
		Chain(frontmatter.New()).
        Chain(markdown.NewCommon()).
        End(dstDir)
    ```

*   Next we want to run our generated HTML through a template to add a header, footer, and a menu; for this we can use
    the [Goldsmith-Layout](https://foosoft.net/projects/goldsmith/plugins/layout/) plugin:

    ```
    goldsmith.Begin(srcDir).
		Chain(frontmatter.New()).
        Chain(markdown.NewCommon()).
        Chain(layout.New("layoutDir/*.html")).
        End(dstDir)
    ```

*   Finally, let's minify our files to reduce data transfer and load times for our site's visitors using the
    [Goldsmith-Minify](https://foosoft.net/projects/goldsmith/plugins/minify/) plugin:

    ```
    goldsmith.Begin(srcDir).
		Chain(frontmatter.New()).
        Chain(markdown.NewCommon()).
        Chain(layout.New("layoutDir/*.html")).
		Chain(minify.New()).
        End(dstDir)
    ```

*   Now that we have all of our plugins chained up, let's look at a complete example which uses the
    [Goldsmith-DevServer](https://foosoft.net/projects/goldsmith/devserver/) library to bootstrap a development sever with live reload:

    ```
    package main

    import (
        "log"

        "github.com/FooSoft/goldsmith"
        "github.com/FooSoft/goldsmith-devserver"
        "github.com/FooSoft/goldsmith-plugins/frontmatter"
        "github.com/FooSoft/goldsmith-plugins/layout"
        "github.com/FooSoft/goldsmith-plugins/livejs"
        "github.com/FooSoft/goldsmith-plugins/markdown"
        "github.com/FooSoft/goldsmith-plugins/minify"
    )

    type builder struct{}

    func (b *builder) Build(srcDir, dstDir string) {
        errs := goldsmith.Begin(srcDir).
            Chain(frontmatter.New()).
            Chain(markdown.NewCommon()).
            Chain(layout.New("layoutDir/*.html")).
            Chain(livejs.New()).
            Chain(minify.New()).
            End(dstDir)

        for _, err := range errs {
            log.Print(err)
        }
    }

    func main() {
        devserver.DevServe(new(builder), 8080, "srcDir", "dstDir")
    }
    ```

I hope that this short series of examples illustrated the inherent simplicity and flexibility of the Goldsmith
pipeline-oriented approach to data processing. Files are injected into the stream at Goldsmith initialization, processed
in parallel through a set of plugins, and are finally written out to disk upon completion.

Files are guaranteed to flow through Goldsmith plugins in the same order, but not necessarily in the same sequence
relative to each other. Timing differences can cause certain files to finish ahead of others; fortunately this, along
with other threading characteristics of the tool, is abstracted from the user. The execution, while appearing to be a
mere series chained methods, will process files taking full advantage of your processor's cores.

## License ##

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
