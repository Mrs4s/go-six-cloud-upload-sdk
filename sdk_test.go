package sixcloudUploader

import (
	"fmt"
	"testing"
)

func TestUploader(t *testing.T) {
	task, err := CreateUploadTask("0a3836b4ef298e7dc9fc5da291252fc4ac3e0c7f:NWI5YzllZWQ4NTJhYjI4ZGMzNTk1MjljMGQ0NjMwZmEwYzRhMzVlNA==:eyJzY29wZSI6Im90aGVyLXN0b3JhZ2U6dXNlci11cGxvYWQvZGlyZWN0LzIwMTktMTItMjcvMTZfMTU3NzQzNDI2MzU5Nzc2Mzk4Ny02NjY2Y2Q3NmY5Njk1NjQ2OWU3YmUzOWQ3NTBjYzdkOS50bXBfaXAiLCJkZWFkbGluZSI6IjE1Nzc1MjA2NjM1OTciLCJvdmVyd3JpdGUiOjEsImNhbGxiYWNrVXJsIjoiaHR0cHM6Ly9hcGkuNnBhbi5jbi92Mi91cGxvYWQvd2NzQ2FsbGJhY2siLCJjYWxsYmFja0JvZHkiOiJmaWxlTmFtZT0yMDE5LTEyLTE0XzAxLTA4LTQwLm1wNCQkJCFRWlNQTElUJCEkJHBhcmVudFBhdGg9LyQkJCFRWlNQTElUJCEkJHNpemU9JChmc2l6ZSkkJCQhUVpTUExJVCQhJCRoYXNoPSQoaGFzaCkkJCQhUVpTUExJVCQhJCRrZXk9JChrZXkpJCQkIVFaU1BMSVQkISQkbWltZVR5cGU9JChtaW1lVHlwZSkkJCQhUVpTUExJVCQhJCRpcD0kKGlwKSQkJCFRWlNQTElUJCEkJGJ1Y2tldD0kKGJ1Y2tldCkkJCQhUVpTUExJVCQhJCR1c2VySWQ9MTYkJCQhUVpTUExJVCQhJCRvcD0wJCQkIVFaU1BMSVQkISQkcGFyZW50SWQ9JCQkIVFaU1BMSVQkISQkdXBsb2FkRmlsZU5hbWU9JChmbmFtZSkkJCQhUVpTUExJVCQhJCRzdGVwPTEkJCQhUVpTUExJVCQhJCR0eXBlPXdjcyIsInNlcGFyYXRlIjoiMCJ9", "https://upload-vod-v1.qiecdn.com", "/Users/sijiang/Downloads/2019-12-14_01-08-40.mp4")
	if err != nil {
		fmt.Println(err)
		return
	}
	client := NewClient(task)
	client.OnUploaded = func(client *UploadClient) {
		fmt.Println("uploaded")
	}
	client.OnUploadFailed = func(client *UploadClient) {
		fmt.Println("upload failed")
	}
	client.BeginUpload()
	<-make(chan bool)
}
