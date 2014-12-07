gost
==========
gost is like a vendoring `go get` that uses Git subtrees. It is useful for
producing repeatable builds and for making coordinates changes across multiple
Go packages hosted on GitHub.

When you `gost get`, any packages that are hosted on GitHub will be sucked in as
subrepositories and any other packages will be included in source form.

Unlike most vendoring mechanisms, gost is not meant to be used within a
subfolder of an existing repo. Rather, to use gost, set up a new project (which
we call a "gost repo") in order to do your vendoring in there.

### Example

#### Setting up a new gost repo

Let's say that we want to make a change to github.com/getlantern/flashlight that
requires changes to various libraries in github.com/getlantern that are used by
flashlight.

##### Install gost

```
go get github.com/getlantern/gost
```

##### Initialize a gost repo

```
mkdir flashlight-build
cd flashlight-build
gost init
```

##### Set the gost repo directory as your GOPATH

```
export GOPATH=`pwd`
```

##### Gost get the main project that we're interested in

```
gost get github.com/getlantern.org/flashlight
```

At this point, we have a gost repo that incorporates flashlight and all of
its dependencies (including test dependencies). We may want to go ahead and
push upstream now.

```
git remote add origin https://github.com/getlantern/flashlight-build.git
git push -u origin master
```

##### Branch from master in preparation for making our changes

```
git checkout -b mybranch master
```

Now we make our changes.

##### Pull in another existing package

Let's say that there's an existing package on GitHub that we need to add to our
GOPATH in order to make this change. We can just `gost get` it.

```
gost get github.com/getlantern/newneededpackage
```

##### Pull in upstream updates

If updates have been made upstream, we can pull these in using `gost get -u`.
It works just like `go get -u` and updates the target package and dependencies.

```
gost get -u github.com/getlantern/flashlight
```

We can even pull in changes from a specific branch

```
gost get -u github.com/getlantern/flashlight specificbranch
```

##### Push our gost get project and submit a PR

```
git push --set-upstream origin mybranch
```

At this point, we can submit a pull request on GitHub, which will show all
changes to all projects in our gost repo (i.e. our GOPATH). Once the PR has
been merged to master, we can pull using git as usual.

##### Contribute changes back upstream to subprojects

```
git checkout master
git pull
gost push github.com/getlantern/flashlight
```

Unlike `gost get` which fetches dependencies, `gost push` only pushes the
specific package indicated in the command.
Note - this `gost push` command is not yet implemented.