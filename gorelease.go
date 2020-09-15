package main

import (
	"flag"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/nwillc/gorelease/config"
	"github.com/nwillc/gorelease/gen/version"
	"golang.org/x/mod/semver"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	flag.Parse()
	if *config.Flags.Version {
		fmt.Printf("version %s\n", version.Version)
		os.Exit(0)
	}
	if *config.Flags.DryRun {
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
		msg := fmt.Sprintf("incorrrect file commit status, %d files, expecting only %s", len(status), config.DotVersionFile)
		if *config.Flags.Dirty {
			log.Println(msg)
		} else {
			panic(fmt.Errorf(msg))
		}
	}

	vs := status.File(config.DotVersionFile)
	if vs.Staging == '?' && vs.Worktree == '?' {
		msg := fmt.Sprintf("%s should be only uncommitted file", config.DotVersionFile)
		if *config.Flags.Dirty {
			log.Println(msg)
		} else {
			panic(fmt.Errorf(msg))
		}
	}
	/*
	 * Get new version.
	 */
	content, err := ioutil.ReadFile(config.DotVersionFile)
	checkIfError("reading .version", err)
	versionStr := strings.Replace(string(content), "\n", "", -1)
	if !semver.IsValid(versionStr) {
		msg := fmt.Sprintf("invalid version %s", versionStr)
		log.Println(msg)
		panic(fmt.Errorf(msg))
	}
	tag := semver.Canonical(versionStr)
	/*
	 * Create the new version file.
	 */
	createVersionGo(*config.Flags.Output, tag)

	if *config.Flags.DryRun {
		os.Exit(0)
	}

	/*
	* Git add the .version and version files.
	 */
	_, err = w.Add(*config.Flags.Output)
	checkIfError(fmt.Sprintf("adding %s", *config.Flags.Output), err)

	_, err = w.Add(config.DotVersionFile)
	checkIfError(fmt.Sprintf("adding %s", config.DotVersionFile), err)

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
		repo.Push(&git.PushOptions{Auth: sshKey})
	}
}

func publicKeys() (*ssh.PublicKeys, error) {
	path, err := os.UserHomeDir()
	checkIfError("finding home directory", err)
	path += "/.ssh/id_rsa"

	publicKey, err := ssh.NewPublicKeysFromFile(config.GitUser, path, "")
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
	contents, err := ioutil.ReadFile(config.LicenseFile)
	if err == nil {
		licenseStr = "/*\n *" + strings.Replace(string(contents), "\n", "\n *", -1) + "\n */"
	}

	versionGo := strings.Replace(versionTemplateStr, "$LICENSE$", licenseStr, 1)
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
		log.Printf("get tags error: %s", err)
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
		log.Printf("iterate tags error: %s", err)
		return false
	}
	return res
}

func setTag(r *git.Repository, tag string) (bool, error) {
	if tagExists(r, tag) {
		log.Printf("tag %s already exists", tag)
		return false, nil
	}
	log.Printf("Set tag %s", tag)
	h, err := r.Head()
	if err != nil {
		log.Printf("get HEAD error: %s", err)
		return false, err
	}
	_, err = r.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
		Tagger:  newSignature(),
		Message: "Release " + tag,
	})

	if err != nil {
		log.Printf("create tag error: %s", err)
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

const versionTemplateStr = `$LICENSE$

package $PACKAGE$

// Code generated by github.com/nwillc/gorelease DO NOT EDIT.

// Version number for official releases.
const Version = "$TAG$"
`
