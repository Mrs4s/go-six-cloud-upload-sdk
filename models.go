package sixcloudUploader

import (
	tsgutils "github.com/typa01/go-utils"
	"os"
)

type (
	UploadTaskInfo struct {
		UploadToken string `json:"uploadToken"`
		UploadUrl   string `json:"uploadUrl"`
		UploadBatch string `json:"uploadBatch"`
		File        string `json:"file"`
		FileSize    int64  `json:"size"`

		Blocks []*UploadTaskBlock `json:"blocks"`
	}

	UploadTaskBlock struct {
		Id          int
		BeginOffset int64
		EndOffset   int64
		Size        int64
		Ctx         string
		Uploaded    bool
		Uploading   bool `json:"-"`

		lastChunkOffset int64  `json:"-"`
		lastChunkCtx    string `json:"-"`
	}

	UploadTaskStatus int
)

const (
	Waiting UploadTaskStatus = iota
	Uploading
	Paused
	Completed
	Failed

	BlockSize int64 = 4 * 1024 * 1024
	ChunkSize int64 = 1024 * 1024
)

func (info *UploadTaskInfo) init() error {
	file, err := os.Stat(info.File)
	if err != nil {
		return err
	}
	info.FileSize = file.Size()
	if info.FileSize <= BlockSize {
		info.Blocks = []*UploadTaskBlock{
			{
				Id:        0,
				EndOffset: info.FileSize,
				Size:      info.FileSize,
			},
		}
		return nil
	}
	var temp int64 = 0
	for temp+BlockSize < info.FileSize {
		info.Blocks = append(info.Blocks, &UploadTaskBlock{
			Id:          len(info.Blocks),
			BeginOffset: temp,
			EndOffset:   temp + BlockSize,
			Size:        BlockSize,
		})
		temp += BlockSize
	}
	info.Blocks = append(info.Blocks, &UploadTaskBlock{
		Id:          len(info.Blocks),
		BeginOffset: temp,
		EndOffset:   info.FileSize,
		Size:        info.FileSize - temp,
	})
	return nil
}

func (info *UploadTaskInfo) UploadedSize() int64 {
	var result int64 = 0
	for _, block := range info.Blocks {
		if block.Uploaded {
			result += block.Size
		}
	}
	return result
}

func (info *UploadTaskInfo) getNextBlockN() int {
	for i, block := range info.Blocks {
		if !block.Uploaded && !block.Uploading {
			return i
		}
	}
	return -1
}

func (info *UploadTaskInfo) allUploaded() bool {
	for _, block := range info.Blocks {
		if !block.Uploaded || block.Uploading {
			return false
		}
	}
	return true
}

func CreateUploadTask(uploadToken, uploadUrl, file string) (*UploadTaskInfo, error) {
	var result = &UploadTaskInfo{
		UploadToken: uploadToken,
		UploadUrl:   uploadUrl,
		UploadBatch: tsgutils.UUID(),
		File:        file,
	}
	return result, result.init()
}
