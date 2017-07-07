// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"fmt"
	"log"
	"os/exec"
	"os/user"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

var (
	sess = session.Must(session.NewSession())
	svc  = iam.New(sess)
)

func getIAMSSHPublicKeys(username string) (publicKeyIds []string) {
	result, err := svc.ListSSHPublicKeys(&iam.ListSSHPublicKeysInput{
		UserName: aws.String(username),
	})

	if err != nil {
		log.Println(err)
	}

	if result.SSHPublicKeys != nil {
		for _, key := range result.SSHPublicKeys {
			result2, err := svc.GetSSHPublicKey(&iam.GetSSHPublicKeyInput{
				Encoding:       aws.String("SSH"),
				UserName:       aws.String(username),
				SSHPublicKeyId: key.SSHPublicKeyId,
			})

			if err != nil {
				log.Println(err)
				continue
			}

			publicKeyIds = append(publicKeyIds, *result2.SSHPublicKey.SSHPublicKeyBody)
		}
	}

	return
}

func syncUser(username string) error {
	sshKeys := getIAMSSHPublicKeys(username)

	if len(sshKeys) == 0 {
		log.Printf("No SSH keys for %s. Skipping.\n", username)
		return nil
	}

	u, err := user.Lookup(username)
	if u == nil {
		log.Println("User already exists", u.Username)
		return nil
	}

	if err != nil {
		log.Println(err)
		return nil
	}

	cmd := exec.Command("adduser", username, "--disabled-password", "--gecos", username)
	if err = cmd.Start(); err != nil {
		return err
	}
	err = cmd.Wait()

	// TODO: Assuming that the group sudo exists.
	cmd = exec.Command("usermod", "--append", "--groups", "sudo", username)
	if err = cmd.Start(); err != nil {
		log.Fatal(err)
	}
	err = cmd.Wait()

	return nil
}

func getIAMUsernames() (usernames []string) {
	result, err := svc.ListUsers(nil)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}

	for _, user := range result.Users {
		usernames = append(usernames, *user.UserName)
	}

	return
}

func main() {
	for _, username := range getIAMUsernames() {
		log.Println("Syncing IAM User:", username)
		if err := syncUser(username); err != nil {
			log.Println(err)
		}
	}
}
