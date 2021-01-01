package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"text/template"

	"github.com/cretz/takecast/pkg/cert"
	"github.com/cretz/takecast/pkg/receiver/mirror"
	"github.com/cretz/takecast/pkg/receiver/webrtc"
	"github.com/cretz/takecast/pkg/server"
	"github.com/spf13/cobra"
)

func recordCmd() *cobra.Command {
	var outFilenameTemplate string
	cmd := applyRun(
		&cobra.Command{
			Use:   "record",
			Short: "Record all incoming streams as webm",
		},
		func(ctx *rootContext) error {
			// Load root CA
			rootCA, err := cert.LoadKeyPairFromFiles(filepath.Join("ca.crt"), filepath.Join("ca.key"))
			if err != nil {
				return fmt.Errorf("failed loading ca.crt/ca.key, did you forget to run 'patch'? err: %w", err)
			}
			// Create recorder
			rec, err := newRecorder(ctx, outFilenameTemplate)
			if err != nil {
				return err
			}
			// Start server listen
			s, err := server.Listen(server.Config{RootCACert: rootCA, Log: ctx.log})
			if err != nil {
				return fmt.Errorf("failed starting server: %w", err)
			}
			defer s.Close()
			// Register mirror application
			if m, err := mirror.New(mirror.Config{Log: ctx.log, OnSession: rec.onSession}); err != nil {
				return fmt.Errorf("failed creating mirror application: %w", err)
			} else if err = s.Receiver.RegisterApplication(m); err != nil {
				return fmt.Errorf("failed registering mirror application: %w", err)
			}
			// Run server in background
			errCh := make(chan error, 1)
			go func() { errCh <- s.Serve() }()
			// Wait for context done or error
			select {
			case <-ctx.Done():
				return nil
			case err := <-errCh:
				return fmt.Errorf("server failed: %w", err)
			}
		},
	)
	cmd.Flags().StringVarP(&outFilenameTemplate, "out-filename-template", "o",
		"./stream-{{.Index}}.webm", "Template to create filename to save each stream as")
	return cmd
}

type recorder struct {
	*rootContext
	filenameTemplate *template.Template
	sessionCounter   int32
}

func newRecorder(ctx *rootContext, filenameTemplate string) (*recorder, error) {
	tmpl, err := template.New("out").Parse(filenameTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed parsing out filename template: %w", err)
	}
	return &recorder{rootContext: ctx, filenameTemplate: tmpl}, nil
}

func (r *recorder) onSession(s *webrtc.Session) {
	// Run it async
	go func() {
		if err := r.runSession(s); err != nil {
			r.log.Warnf("recorder failure: %v", err)
		}
	}()
}

func (r *recorder) runSession(s *webrtc.Session) error {
	var filename strings.Builder
	err := r.filenameTemplate.Execute(&filename, map[string]interface{}{
		"Index": atomic.AddInt32(&r.sessionCounter, 1),
	})
	if err != nil {
		return fmt.Errorf("failed executing filename template: %w", err)
	}
	// Create/overwrite file
	r.log.Infof("Recording new stream to %v", filename)
	w, err := os.OpenFile(filename.String(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed creating file at %v: %w", filename, err)
	}
	return webrtc.SaveSessionToWebM(r, w, s)
}
