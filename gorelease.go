package main

import (
	"flag"
	"fmt"
	"github.com/blang/semver/v4"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/nwillc/gorelease/version"
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

var flags struct {
	dryRun  *bool
	dirty   *bool
	version *bool
}

func init() {
	flags.dryRun = flag.Bool("dryrun", false, "Perform a dry run, no files changed or tags/files pushed.")
	flags.dirty = flag.Bool("dirty", false, "Allow dirty repository with uncommitted files.")
	flags.version = flag.Bool("version", false, "Display version.")
}

func main() {
	flag.Parse()
	if *flags.version {
		fmt.Printf("version %s\n", version.Version)
		os.Exit(0)
	}
	if *flags.dryRun {
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
		msg := fmt.Sprintf("incorrrect file commit status, %d files, expecting only %s", len(status), dotVersionFile)
		if *flags.dirty {
			log.Println(msg)
		} else {
			panic(fmt.Errorf(msg))
		}
	}

	vs := status.File(dotVersionFile)
	if vs.Staging == '?' && vs.Worktree == '?' {
		msg := fmt.Sprintf("%s should be only uncommitted file", dotVersionFile)
		if *flags.dirty {
			log.Println(msg)
		} else {
			panic(fmt.Errorf(msg))
		}
	}
	/*
	 * Get new version.
	 */
	content, err := ioutil.ReadFile(dotVersionFile)
	checkIfError("reading .version", err)
	versionStr := strings.Replace(string(content), "\n", "", -1)
	v, err := semver.Make(versionStr)
	checkIfError("parsing version", err)
	tag := "v" + v.String()

	/*
	 * Create the new version file.
	 */
	createVersionGo(output, tag)

	if *flags.dryRun {
		os.Exit(0)
	}

	/*
	* Git add the .version and version files.
	 */
	_, err = w.Add(output)
	checkIfError("adding version.go", err)

	_, err = w.Add(dotVersionFile)
	checkIfError("adding .version", err)

	/*
	* Git commit the files.
	 */
	_, err = w.Commit("Generated for "+tag, &git.CommitOptions{
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

	publicKey, err := ssh.NewPublicKeysFromFile(gitUser, path, "")
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
	contents, err := ioutil.ReadFile(licenseFile)
	if err == nil {
		licenseStr = "/*\n *" + strings.Replace(string(contents), "\n", "\n *", -1) + "\n */"
	}

	versionGo := strings.Replace(versionTemplateStr, "$LICENSE$", licenseStr, 1)
	versionGo = strings.Replace(versionGo, "$TAG$", tag, 1)

	if *flags.dryRun {
		fmt.Println(versionGo)
		return
	}

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

package version

// Version number for official releases. Updated with gorelease.
const Version = "$TAG$"
`
