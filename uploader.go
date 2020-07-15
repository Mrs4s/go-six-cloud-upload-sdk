package sixcloudUploader

import (
	"bytes"
	"github.com/tidwall/gjson"
	"io"
	"net/http"
	"os"
	"strconv"
)

type UploadClient struct {
	Info           *UploadTaskInfo
	ThreadCount    int
	Status         UploadTaskStatus
	MaxRetry       int
	OnUploadFailed func(*UploadClient)
	OnUploaded     func(*UploadClient)
	LogAction      func(string)

	client *http.Client
}

func NewClient(info *UploadTaskInfo, threadCount ...int) *UploadClient {
	result := &UploadClient{
		Info:        info,
		Status:      Waiting,
		ThreadCount: 1,
		MaxRetry:    10,
		client: &http.Client{
			Transport: &http.Transport{
				MaxConnsPerHost:     0,
				MaxIdleConns:        0,
				MaxIdleConnsPerHost: 999,
			},
		},
	}
	if len(threadCount) == 1 && threadCount[0] > 0 {
		result.ThreadCount = threadCount[0]
	}
	return result
}

func (cli *UploadClient) BeginUpload() {
	cli.Status = Uploading
	ch := make(chan bool)
	go func() {
		for {
			su := <-ch
			if !su {
				for _, block := range cli.Info.Blocks {
					block.Uploading = false
				}
				if cli.OnUploadFailed != nil {
					cli.OnUploadFailed(cli)
				}
				return
			}
			block := cli.Info.getNextBlockN()
			if block != -1 {
				cli.Info.Blocks[block].Uploading = true
				go cli.Info.Blocks[block].createBlock(cli, ch)
				continue
			}
			if cli.Info.allUploaded() {
				cli.mkfile()
				if cli.OnUploaded != nil {
					cli.OnUploaded(cli)
				}
				break
			}
		}
	}()
	for i := 0; i < cli.ThreadCount; i++ {
		block := cli.Info.getNextBlockN()
		if block != -1 {
			cli.Info.Blocks[block].Uploading = true
			go cli.Info.Blocks[block].createBlock(cli, ch)
		}
	}
}

func (block *UploadTaskBlock) createBlock(cli *UploadClient, ch chan bool, retry ...int) {
	rt := 0
	if len(retry) > 0 {
		rt = retry[0]
	}
	if rt > cli.MaxRetry {
		ch <- false
		return
	}
	defer func() {
		if pan := recover(); pan != nil {
			if cli.LogAction != nil {
				cli.LogAction(pan.(string))
			}
			go block.createBlock(cli, ch, rt+1)
		}
	}()
	file, err := os.OpenFile(cli.Info.File, os.O_RDONLY, 0666)
	if err != nil {
		panic("can't open file " + cli.Info.File)
	}
	defer file.Close()
	_, _ = file.Seek(block.BeginOffset, 0)
	buff := make([]byte, ChunkSize)
	flen, _ := file.Read(buff)
	if int64(flen) < ChunkSize {
		buff = buff[:flen]
	}
	req, err := http.NewRequest("POST",
		cli.Info.UploadUrl+"/mkblk/"+strconv.FormatInt(block.Size, 10)+"/"+strconv.FormatInt(int64(block.Id), 10),
		bytes.NewReader(buff),
	)
	if err != nil {
		panic("cannot create request.")
	}
	req.ContentLength = int64(flen)
	req.Header["Content-Type"] = []string{"application/octet-stream"}
	req.Header["Authorization"] = []string{cli.Info.UploadToken}
	req.Header["UploadBatch"] = []string{cli.Info.UploadBatch}
	resp, err := cli.client.Do(req)
	if err != nil {
		panic("send request failed.")
	}
	buff = make([]byte, 1024*20)
	if resp.StatusCode < 200 || resp.StatusCode > 300 {
		panic("http response not be successful.")
	}
	i, err := resp.Body.Read(buff)
	if err != nil && err != io.EOF {
		panic("read response stream failed.")
	}
	token := gjson.ParseBytes(buff[:i])
	if token.Get("code").Exists() {
		panic("server response not be successful.")
	}
	block.Ctx = token.Get("ctx").Str
	block.lastChunkCtx = token.Get("ctx").Str
	block.lastChunkOffset = token.Get("offset").Int()
	if int64(flen) < ChunkSize {
		block.Uploaded = true
		block.Uploading = false
		ch <- true
		return
	}
	block.chunkUpload(cli, ch)
}

func (block *UploadTaskBlock) chunkUpload(cli *UploadClient, ch chan bool, retry ...int) {
	defer func() {
		if pan := recover(); pan != nil {
			if cli.LogAction != nil {
				cli.LogAction(pan.(string))
			}
			rt := 1
			if len(retry) > 0 {
				rt = retry[0] + 1
			}
			go block.createBlock(cli, ch, rt)
		}
	}()
	for block.Uploading {
		file, err := os.OpenFile(cli.Info.File, os.O_RDONLY, 0666)
		if err != nil {
			panic("can't open file " + cli.Info.File)
		}
                defer file.Close()
		_, _ = file.Seek(block.BeginOffset+block.lastChunkOffset, 0)
		var buffLen int64
		if block.BeginOffset+block.lastChunkOffset+ChunkSize > block.EndOffset {
			buffLen = block.EndOffset - (block.BeginOffset + block.lastChunkOffset)
		} else {
			buffLen = ChunkSize
		}
		buff := make([]byte, buffLen)
		flen, _ := file.Read(buff)
		if flen == 0 {
			block.Uploaded = true
			block.Uploading = false
			ch <- true
			return
		}
		req, err := http.NewRequest("POST",
			cli.Info.UploadUrl+"/bput/"+block.lastChunkCtx+"/"+strconv.FormatInt(block.lastChunkOffset, 10),
			bytes.NewReader(buff),
		)
		if err != nil {
			panic("cannot create request.")
		}
		req.ContentLength = int64(flen)
		req.Header["Content-Type"] = []string{"application/octet-stream"}
		req.Header["Authorization"] = []string{cli.Info.UploadToken}
		req.Header["UploadBatch"] = []string{cli.Info.UploadBatch}
		resp, err := cli.client.Do(req)
		if err != nil {
			panic("send request failed.")
		}
		buff = make([]byte, 1024*20)
		if resp.StatusCode < 200 || resp.StatusCode > 300 {
			panic("http response not be successful.")
		}
		i, err := resp.Body.Read(buff)
		if err != nil && err != io.EOF {
			panic("read response stream failed.")
		}
		token := gjson.ParseBytes(buff[:i])
		if token.Get("code").Exists() {
			panic("server response not be successful.")
		}
		block.Ctx = token.Get("ctx").Str
		block.lastChunkCtx = token.Get("ctx").Str
		block.lastChunkOffset = token.Get("offset").Int()
	}
	ch <- false
}

func (cli *UploadClient) mkfile(retry ...int) (context string, ok bool) {
	rt := 0
	if len(retry) > 0 {
		rt = retry[0]
	}
	defer func() {
		if pan := recover(); pan != nil {
			if rt > cli.MaxRetry {
				context = ""
				ok = false
				return
			}
			cli.mkfile(rt + 1)
		}
	}()
	if rt > cli.MaxRetry {
		return "", false
	}
	if cli.Info.allUploaded() {
		ctxs := cli.Info.Blocks[0].Ctx
		for _, block := range cli.Info.Blocks[1:] {
			ctxs += "," + block.Ctx
		}
		req, err := http.NewRequest("POST", cli.Info.UploadUrl+"/mkfile/"+strconv.FormatInt(cli.Info.FileSize, 10), bytes.NewReader([]byte(ctxs)))
		if err != nil {
			panic("cannot create request.")
		}
		req.Header["Content-Type"] = []string{"text/plain;charset=UTF-8"}
		req.Header["Authorization"] = []string{cli.Info.UploadToken}
		req.Header["UploadBatch"] = []string{cli.Info.UploadBatch}
		resp, err := cli.client.Do(req)
		if err != nil {
			panic("send request failed.")
		}
		buff := make([]byte, 10*1024)
		if resp.StatusCode < 200 || resp.StatusCode > 300 {
			panic("http response not be successful.")
		}
		i, err := resp.Body.Read(buff)
		if err != nil && err != io.EOF {
			panic("read response stream failed.")
		}
		token := gjson.ParseBytes(buff[:i])
		respToken := gjson.Parse(token.Get("response").Str)
		return respToken.Get("hash").Str, true
	}
	return "", false
}
