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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/nwillc/gorelease/gen/version"
	"github.com/nwillc/gorelease/setup"
	"github.com/nwillc/gorelease/utils"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
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

	// Get the repo
	repo := utils.GetRepository("")
	w, err := repo.Worktree()
	utils.CheckIfError("repository access", err)

	// Check that we are ready for release.
	if err := repoReady(w); err != nil {
		panic(err)
	}

	// Get new version.
	tag := getNewVersion()

	/*
	 * Create the new version file.
	 */
	createVersionGo(*setup.Flags.Output, tag)

	if *setup.Flags.DryRun {
		os.Exit(setup.NormalExit)
	}

	// Git add the .version and version files.
	addVersionFiles(w)

	/*
	* Git commit the files.
	 */
	_, err = w.Commit(fmt.Sprintf("Updated for release %s", tag), &git.CommitOptions{
		Author: utils.NewSignature(),
	})
	utils.CheckIfError("committing files", err)

	/*
	* Git create new tag.
	 */
	ok, err := setTag(repo, tag)
	utils.CheckIfError("setting tag", err)

	if !ok {
		panic(fmt.Errorf("unable to set tag %s", tag))
	}

	sshKey, _ := utils.PublicKeys()

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
			if *setup.Flags.Verbose {
				log.Println(err)
			}
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

func addVersionFiles(w *git.Worktree) {
	_, err := w.Add(*setup.Flags.Output)
	utils.CheckIfError(fmt.Sprintf("adding %s", *setup.Flags.Output), err)

	_, err = w.Add(setup.DotVersionFile)
	utils.CheckIfError(fmt.Sprintf("adding %s", setup.DotVersionFile), err)
}

func getNewVersion() string {
	content, err := ioutil.ReadFile(setup.DotVersionFile)
	utils.CheckIfError("reading .version", err)
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
		utils.CheckIfError("unable to read go.mod", err)
		modFile, err := modfile.Parse(setup.ModuleFile, content, nil)
		utils.CheckIfError("unable to parse go.mod", err)
		if !strings.HasSuffix(modFile.Module.Mod.Path, semver.Major(tag)) {
			log.Printf("Major version specified (%s) not found at end of go.mod module %s", semver.Major(tag), modFile.Module.Mod.Path)
			os.Exit(setup.VersionConflict)
		}
	}
	return tag
}

func repoReady(w *git.Worktree) error {
	status, err := w.Status()
	utils.CheckIfError("repository status", err)

	if len(status) != 1 {
		msg := fmt.Sprintf("incorrrect file commit status, %d files, expecting only %s", len(status), setup.DotVersionFile)
		if *setup.Flags.Dirty {
			log.Println("dirty override: ", msg)
		} else {
			return fmt.Errorf(msg)
		}
	}

	vs := status.File(setup.DotVersionFile)
	if vs.Staging == '?' && vs.Worktree == '?' {
		msg := fmt.Sprintf("%s should be only uncommitted file", setup.DotVersionFile)
		if *setup.Flags.Dirty {
			log.Println("dirty override: ", msg)
		} else {
			return fmt.Errorf(msg)
		}
	}
	return nil
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
		utils.CheckIfError("Unable to create", err)
	}

	packageName := filepath.Base(path)
	if packageName == "." {
		packageName = "main"
	}
	versionGo = strings.Replace(versionGo, "$PACKAGE$", packageName, -1)

	f, err := os.Create(fileName)
	utils.CheckIfError("creating version.go", err)
	defer f.Close()

	_, err = f.WriteString(versionGo)
	utils.CheckIfError("writing version.go", err)
}

func setTag(r *git.Repository, tag string) (bool, error) {
	if utils.TagExists(r, tag) {
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
		Tagger:  utils.NewSignature(),
		Message: "Release " + tag,
	})

	if err != nil {
		log.Println("create tag error:", err)
		return false, err
	}

	return true, nil
}
