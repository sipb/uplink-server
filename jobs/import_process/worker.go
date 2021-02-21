// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package import_process

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/mattermost/mattermost-server/v5/app"
	"github.com/mattermost/mattermost-server/v5/jobs"
	tjobs "github.com/mattermost/mattermost-server/v5/jobs/interfaces"
	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/utils"
)

func init() {
	app.RegisterJobsImportProcessInterface(func(a *app.App) tjobs.ImportProcessInterface {
		return &ImportProcessInterfaceImpl{a}
	})
}

type ImportProcessInterfaceImpl struct {
	app *app.App
}

type ImportProcessWorker struct {
	name        string
	stopChan    chan struct{}
	stoppedChan chan struct{}
	jobsChan    chan model.Job
	jobServer   *jobs.JobServer
	app         *app.App
}

func (i *ImportProcessInterfaceImpl) MakeWorker() model.Worker {
	return &ImportProcessWorker{
		name:        "ImportProcess",
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
		jobsChan:    make(chan model.Job),
		jobServer:   i.app.Srv().Jobs,
		app:         i.app,
	}
}

func (w *ImportProcessWorker) JobChannel() chan<- model.Job {
	return w.jobsChan
}

func (w *ImportProcessWorker) Run() {
	mlog.Debug("Worker started", mlog.String("worker", w.name))

	defer func() {
		mlog.Debug("Worker finished", mlog.String("worker", w.name))
		close(w.stoppedChan)
	}()

	for {
		select {
		case <-w.stopChan:
			mlog.Debug("Worker received stop signal", mlog.String("worker", w.name))
			return
		case job := <-w.jobsChan:
			mlog.Debug("Worker received a new candidate job.", mlog.String("worker", w.name))
			w.doJob(&job)
		}
	}
}

func (w *ImportProcessWorker) Stop() {
	mlog.Debug("Worker stopping", mlog.String("worker", w.name))
	close(w.stopChan)
	<-w.stoppedChan
}

func (w *ImportProcessWorker) doJob(job *model.Job) {
	if claimed, err := w.jobServer.ClaimJob(job); err != nil {
		mlog.Warn("Worker experienced an error while trying to claim job",
			mlog.String("worker", w.name),
			mlog.String("job_id", job.Id),
			mlog.String("error", err.Error()))
		return
	} else if !claimed {
		return
	}

	importFileName, ok := job.Data["import_file"]
	if !ok {
		appError := model.NewAppError("ImportProcessWorker", "import_process.worker.do_job.missing_file", nil, "", http.StatusBadRequest)
		w.setJobError(job, appError)
		return
	}

	importFilePath := filepath.Join(*w.app.Config().ImportSettings.Directory, importFileName)
	if ok, err := w.app.FileExists(importFilePath); err != nil {
		w.setJobError(job, err)
		return
	} else if !ok {
		appError := model.NewAppError("ImportProcessWorker", "import_process.worker.do_job.file_exists", nil, "", http.StatusBadRequest)
		w.setJobError(job, appError)
		return
	}

	importFileSize, appErr := w.app.FileSize(importFilePath)
	if appErr != nil {
		w.setJobError(job, appErr)
		return
	}

	importFile, appErr := w.app.FileReader(importFilePath)
	if appErr != nil {
		w.setJobError(job, appErr)
		return
	}
	defer importFile.Close()

	// TODO (MM-30187): improve this process by eliminating the need to unzip the import
	// file locally and instead do the whole bulk import process in memory by
	// streaming the import file.

	// create a temporary dir to extract the zipped import file.
	dir, err := ioutil.TempDir("", "import")
	if err != nil {
		appError := model.NewAppError("ImportProcessWorker", "import_process.worker.do_job.tmp_dir", nil, err.Error(), http.StatusInternalServerError)
		w.setJobError(job, appError)
		return
	}
	defer os.RemoveAll(dir)

	// extract the contents of the zipped file.
	paths, err := utils.UnzipToPath(importFile.(io.ReaderAt), importFileSize, dir)
	if err != nil {
		appError := model.NewAppError("ImportProcessWorker", "import_process.worker.do_job.unzip", nil, err.Error(), http.StatusInternalServerError)
		w.setJobError(job, appError)
		return
	}

	// find JSONL import file.
	var jsonFilePath string
	for _, path := range paths {
		if filepath.Ext(path) == ".jsonl" {
			jsonFilePath = path
			break
		}
	}

	if jsonFilePath == "" {
		appError := model.NewAppError("ImportProcessWorker", "import_process.worker.do_job.missing_jsonl", nil, "", http.StatusBadRequest)
		w.setJobError(job, appError)
		return
	}

	jsonFile, err := os.Open(jsonFilePath)
	if err != nil {
		appError := model.NewAppError("ImportProcessWorker", "import_process.worker.do_job.open_file", nil, err.Error(), http.StatusInternalServerError)
		w.setJobError(job, appError)
		return
	}

	// do the actual import.
	appErr, lineNumber := w.app.BulkImportWithPath(jsonFile, false, runtime.NumCPU(), filepath.Join(dir, app.ExportDataDir))
	if appErr != nil {
		job.Data["line_number"] = strconv.Itoa(lineNumber)
		w.setJobError(job, appErr)
		return
	}

	// remove import file when done.
	if appErr := w.app.RemoveFile(importFilePath); appErr != nil {
		w.setJobError(job, appErr)
		return
	}

	mlog.Info("Worker: Job is complete", mlog.String("worker", w.name), mlog.String("job_id", job.Id))
	w.setJobSuccess(job)
}

func (w *ImportProcessWorker) setJobSuccess(job *model.Job) {
	if err := w.app.Srv().Jobs.SetJobSuccess(job); err != nil {
		mlog.Error("Worker: Failed to set success for job", mlog.String("worker", w.name), mlog.String("job_id", job.Id), mlog.String("error", err.Error()))
		w.setJobError(job, err)
	}
}

func (w *ImportProcessWorker) setJobError(job *model.Job, appError *model.AppError) {
	if err := w.app.Srv().Jobs.SetJobError(job, appError); err != nil {
		mlog.Error("Worker: Failed to set job error", mlog.String("worker", w.name), mlog.String("job_id", job.Id), mlog.String("error", err.Error()))
	}
}
