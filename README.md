# Goldsmith #

Goldsmith is a static website generator developed in Go with flexibility, extensibility, and performance as primary
design considerations. With Goldsmith you can easily build and deploy any type of site, whether it is a personal blog,
image gallery, or a corporate homepage; the tool no assumptions are made about your layout or file structure. Goldsmith
is trivially extensible via a plugin architecture which makes it simple to perform complex data transformations
concurrently. A growing set of core plugins, [Goldsmith-Plugins](https://foosoft.net/projects/goldsmith-plugins/), is provided to make it
easier to get started with this tool to generate static websites.

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
*   A full [Node.js](https://nodejs.org/) is stack (including dependencies) is required to build sites.

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

## Usage ##

Goldsmith is at it's core, a pipeline-based file processor. Files are loaded from the source directory, processed by any
number of plugins, and are finally output to the destination directory. Rather than explaining the process in detail
conceptually, I will show some code samples which show how this tool can be used in practice.

*   Start by copying files from a source directory to a destination directory (simplest possible use case):
    ```
    goldsmith.Begin(srcDir).
        End(dstDir)
    ```

*   Now let's also convert our [Markdown](https://daringfireball.net/projects/markdown/) files to HTML using the
    [markdown plugin](https://foosoft.net/projects/goldsmith-plugins/markdown):
    ```
    goldsmith.Begin(srcDir).
        Chain(markdown.NewCommon()).
        End(dstDir)
    ```

*   If we are using *front matter* in our Markdown files, we can easily extract it by using the
    [frontmatter plugin](https://foosoft.net/projects/goldsmith-plugins/frontmatter):
    ```
    goldsmith.Begin(srcDir).
		Chain(frontmatter.New()).
        Chain(markdown.NewCommon()).
        End(dstDir)
    ```

*   Next we want to run our generated HTML through a template to add a header, footer, and a menu; for this we
    can use the [layout plugin](https://foosoft.net/projects/goldsmith-plugins/layout):
    ```
    goldsmith.Begin(srcDir).
		Chain(frontmatter.New()).
        Chain(markdown.NewCommon()).
		Chain(layout.New(
            layoutFiles,        // array of paths for files containing template definitions
            templateNameVar,    // metadata variable that contains the name of the template to use
            contentStoreVar,    // metadata variable configured in template to insert content
            defTemplateName,    // name of a default template to use if one is not specified
            userFuncs,          // mapping of functions which can be executed from templates
		)).
        End(dstDir)
    ```

*   Finally, let's [minify](https://en.wikipedia.org/wiki/Minification_(programming)) our files to reduce data transfer
    and load times for our site's visitors using the [minify plugin](https://foosoft.net/projects/goldsmith-plugins/minify).
    ```
    goldsmith.Begin(srcDir).
		Chain(frontmatter.New()).
        Chain(markdown.NewCommon()).
		Chain(layout.New(layoutFiles, templateNameVar, contentStoreVar, defTemplateName, userFuncs)).
		Chain(minify.New()).
        End(dstDir)
    ```

I hope this simple example effectively illustrates the conceptual simplicity of the Goldsmith pipeline-based processing
method. Files are injected into the stream at Goldsmith initialization, processed in parallel through a series of
plugins, and are finally written out to disk upon completion.

Files are guaranteed to flow through Goldsmith plugins in the same order, but not necessarily in the same sequence
relative to each other. Timing differences can cause certain files to finish ahead of others; fortunately this, along
with other threading characteristics of the tool is abstracted from the user. The execution, while appearing to be a
mere series chained methods, will process files using all of your system's cores.

## License ##

MIT
