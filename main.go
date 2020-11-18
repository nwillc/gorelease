/*
 * Copyright (c) 2020,  nwillc@gmail.com
 *
 * Permission to use, copy, modify, and/or distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 *
 */

package main

import (
	"flag"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/nwillc/gorelease/gen/version"
	"github.com/nwillc/gorelease/setup"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

const v2 = "v2.0.0"

func main() {
	flag.Parse()
	if *setup.Flags.Version {
		fmt.Printf("version %s\n", version.Version)
		os.Exit(setup.NormalExit)
	}
	if *setup.Flags.DryRun {
		log.Println("Performing dry run.")
	}
	repo := getRepository("")
	w, err := repo.Worktree()
	checkIfError("repository access", err)
	status, err := w.Status()
	checkIfError("repository status", err)

	/*
	 * Check that we are ready for release.
	 */
	if len(status) != 1 {
		msg := fmt.Sprintf("incorrrect file commit status, %d files, expecting only %s", len(status), setup.DotVersionFile)
		if *setup.Flags.Dirty {
			log.Println(msg)
		} else {
			panic(fmt.Errorf(msg))
		}
	}

	vs := status.File(setup.DotVersionFile)
	if vs.Staging == '?' && vs.Worktree == '?' {
		msg := fmt.Sprintf("%s should be only uncommitted file", setup.DotVersionFile)
		if *setup.Flags.Dirty {
			log.Println(msg)
		} else {
			panic(fmt.Errorf(msg))
		}
	}
	/*
	 * Get new version.
	 */
	content, err := ioutil.ReadFile(setup.DotVersionFile)
	checkIfError("reading .version", err)
	versionStr := strings.Replace(string(content), "\n", "", -1)
	if !semver.IsValid(versionStr) {
		msg := fmt.Sprintf("invalid version %s", versionStr)
		log.Println(msg)
		panic(fmt.Errorf(msg))
	}
	tag := semver.Canonical(versionStr)

	if semver.Compare(tag, v2) >= 0 {
		log.Printf("Warning %s >= %s", tag, v2)
		content, err := ioutil.ReadFile(setup.ModuleFile)
		checkIfError("unable to read go.mod", err)
		modFile, err := modfile.Parse(setup.ModuleFile, content, nil)
		checkIfError("unable to parse go.mod", err)
		if !strings.HasSuffix(modFile.Module.Mod.Path, semver.Major(tag)) {
			log.Printf("Major version specified (%s) not found at end of go.mod module %s", semver.Major(tag), modFile.Module.Mod.Path)
			os.Exit(setup.VersionConflict)
		}
	}

	/*
	 * Create the new version file.
	 */
	createVersionGo(*setup.Flags.Output, tag)

	if *setup.Flags.DryRun {
		os.Exit(setup.NormalExit)
	}

	/*
	* Git add the .version and version files.
	 */
	_, err = w.Add(*setup.Flags.Output)
	checkIfError(fmt.Sprintf("adding %s", *setup.Flags.Output), err)

	_, err = w.Add(setup.DotVersionFile)
	checkIfError(fmt.Sprintf("adding %s", setup.DotVersionFile), err)

	/*
	* Git commit the files.
	 */
	_, err = w.Commit(fmt.Sprintf("Updated for release %s", tag), &git.CommitOptions{
		Author: newSignature(),
	})
	checkIfError("committing files", err)

	/*
	* Git create new tag.
	 */
	ok, err := setTag(repo, tag)
	checkIfError("setting tag", err)

	if !ok {
		panic(fmt.Errorf("unable to set tag %s", tag))
	}

	sshKey, _ := publicKeys()

	/*
	 * Git push the tag
	 */
	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{"refs/tags/*:refs/tags/*"},
	})
	if err != nil && sshKey != nil {
		err = repo.Push(&git.PushOptions{
			RemoteName: "origin",
			RefSpecs:   []config.RefSpec{"refs/tags/*:refs/tags/*"},
			Auth:       sshKey,
		})
		if err != nil {
			log.Printf("Push failed, please: git push origin %s; git push", tag)
		}
	}

	/*
	 * Push the entire repo
	 */
	err = repo.Push(&git.PushOptions{})
	if err != nil {
		_ = repo.Push(&git.PushOptions{Auth: sshKey})
	}
}

func publicKeys() (*ssh.PublicKeys, error) {
	path, err := os.UserHomeDir()
	checkIfError("finding home directory", err)
	path += "/.ssh/id_rsa"

	publicKey, err := ssh.NewPublicKeysFromFile(setup.GitUser, path, "")
	if err != nil {
		return nil, err
	}
	return publicKey, nil
}

func newSignature() *object.Signature {
	userInfo, err := user.Current()
	checkIfError("getting current user", err)
	sig := object.Signature{
		Name: userInfo.Name,
		When: time.Now(),
	}
	return &sig
}

func createVersionGo(fileName string, tag string) {
	licenseStr := ""
	for _, licenseFile := range strings.Fields(setup.LicenseFiles) {
		contents, err := ioutil.ReadFile(licenseFile)
		if err == nil {
			licenseStr = "/*\n *" + strings.Replace(string(contents), "\n", "\n *", -1) + "\n */\n"
			break
		}
	}

	versionGo := strings.Replace(setup.VersionTemplateStr, "$LICENSE$", licenseStr, 1)
	versionGo = strings.Replace(versionGo, "$TAG$", tag, 1)

	path := filepath.Dir(fileName)
	folderInfo, err := os.Stat(path)
	if os.IsNotExist(err) || !folderInfo.IsDir() {
		log.Printf("Folder %s does not exist - creating\n", path)
		err := os.MkdirAll(path, 0755)
		checkIfError("Unable to create", err)
	}

	packageName := filepath.Base(path)
	if packageName == "." {
		packageName = "main"
	}
	versionGo = strings.Replace(versionGo, "$PACKAGE$", packageName, -1)

	f, err := os.Create(fileName)
	checkIfError("creating version.go", err)
	defer f.Close()

	_, err = f.WriteString(versionGo)
	checkIfError("writing version.go", err)
}

func getRepository(repo string) *git.Repository {
	if repo == "" {
		repo = "."
	}
	r, err := git.PlainOpenWithOptions(repo, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		panic(err)
	}
	return r
}

func tagExists(r *git.Repository, tag string) bool {
	tagFoundErr := "tag was found"
	tags, err := r.Tags()
	if err != nil {
		log.Println("get tags error:", err)
		return false
	}
	res := false
	err = tags.ForEach(func(t *plumbing.Reference) error {
		if strings.HasSuffix(t.Name().String(), tag) {
			res = true
			return fmt.Errorf(tagFoundErr)
		}
		return nil
	})
	if err != nil && err.Error() != tagFoundErr {
		log.Println("iterate tags error:", err)
		return false
	}
	return res
}

func setTag(r *git.Repository, tag string) (bool, error) {
	if tagExists(r, tag) {
		log.Printf("tag %s already exists", tag)
		return false, nil
	}
	log.Println("Set tag ", tag)
	h, err := r.Head()
	if err != nil {
		log.Println("get HEAD error:", err)
		return false, err
	}
	_, err = r.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
		Tagger:  newSignature(),
		Message: "Release " + tag,
	})

	if err != nil {
		log.Println("create tag error:", err)
		return false, err
	}

	return true, nil
}

func checkIfError(msg string, err error) {
	if err == nil {
		return
	}

	panic(fmt.Errorf("%v: %v", msg, err))
}