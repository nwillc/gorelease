package main

import (
	"fmt"
	"github.com/blang/semver/v4"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strings"
	"time"
)

const (
	output         = "version/version.go"
	dotVersionFile = ".version"
	licenseFile    = "LICENSE.md"
	gitUser        = "git"
)

func main() {
	repo := getRepository("")
	w, err := repo.Worktree()
	checkIfError(err)
	status, err := w.Status()
	checkIfError(err)

	/*
	 * Check that we are ready for release.
	 */
	if len(status) != 1 {
		panic(fmt.Errorf("incorrrect file commit status, %d files, expecting only %s", len(status), dotVersionFile))
	}

	vs := status.File(dotVersionFile)
	if vs.Staging == '?' && vs.Worktree == '?' {
		panic(fmt.Errorf("%s should be only uncommitted file", dotVersionFile))
	}
	/*
	 * Get new version.
	 */
	content, err := ioutil.ReadFile(dotVersionFile)
	checkIfError(err)
	versionStr := strings.Replace(string(content), "\n", "", -1)
	v, err := semver.Make(versionStr)
	checkIfError(err)
	tag := "v" + v.String()

	/*
	 * Create the new version file.
	 */
	createVersionGo(output, tag)

	/*
	* Git add the .version and version files.
	 */
	_, err = w.Add(output)
	checkIfError(err)

	_, err = w.Add(dotVersionFile)
	checkIfError(err)

	/*
	* Git commit the files.
	 */
	_, err = w.Commit("Generated for "+tag, &git.CommitOptions{
		Author: newSignature(),
	})
	checkIfError(err)

	/*
	* Git create new tag.
	 */
	ok, err := setTag(repo, tag)
	checkIfError(err)

	if !ok {
		panic(fmt.Errorf("unable to set tag %s", tag))
	}

	/*
	 * Git push the tag
	 */
	err = repo.Push(newPushOptions([]config.RefSpec{"refs/tags/*:refs/tags/*"}, nil))
	if err != nil {
		sshKey, _ := publicKeys()
		err = repo.Push(newPushOptions([]config.RefSpec{"refs/tags/*:refs/tags/*"}, sshKey))
		if err != nil {
			log.Printf("Push failed, please: git push origin %s; git push", tag)
		}
	}

	/*
	 * Push the entire repo
	 */
	err = repo.Push(newPushOptions(nil, nil))
	if err != nil {
		sshKey, _ := publicKeys()
		repo.Push(newPushOptions(nil, sshKey))
	}
}

func newPushOptions(refSpecs []config.RefSpec, keys *ssh.PublicKeys) *git.PushOptions {
	return &git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   refSpecs,
		Auth:       keys,
	}
}

func publicKeys() (*ssh.PublicKeys, error) {
	path, err := os.UserHomeDir()
	checkIfError(err)
	path += "/.ssh/id_rsa"

	publicKey, err := ssh.NewPublicKeysFromFile(gitUser, path, "")
	if err != nil {
		return nil, err
	}
	return publicKey, nil
}

func newSignature() *object.Signature {
	userInfo, err := user.Current()
	checkIfError(err)
	sig := object.Signature{
		Name: userInfo.Name,
		When: time.Now(),
	}
	return &sig
}

func createVersionGo(fileName string, tag string) {
	contents, err := ioutil.ReadFile(licenseFile)
	checkIfError(err)
	licenseStr := strings.Replace(string(contents), "\n", "\n *", -1)

	versionGo := strings.Replace(versionTemplateStr, "$LICENSE$", licenseStr, 1)
	versionGo = strings.Replace(versionGo, "$TAG$", tag, 1)

	f, err := os.Create(fileName)
	checkIfError(err)
	defer f.Close()

	_, err = f.WriteString(versionGo)
	checkIfError(err)
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

func checkIfError(err error) {
	if err == nil {
		return
	}

	panic(err)
}

const versionTemplateStr = `/*
 * $LICENSE$
 */

package version

// Version number for official releases updated with go generate.
var Version = "$TAG$"
`
