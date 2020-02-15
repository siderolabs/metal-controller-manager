package tftp

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pin/tftp"
)

// readHandler is called when client starts file download from server
func readHandler(filename string, rf io.ReaderFrom) error {
	// TODO(andrewrynhard): Clean the file name path.
	filename = filepath.Join("/var/lib/arges/tftp", filename)

	file, err := os.Open(filename)
	if err != nil {
		log.Printf("%v", err)

		return err
	}

	n, err := rf.ReadFrom(file)
	if err != nil {
		log.Printf("%v", err)

		return err
	}

	log.Printf("%d bytes sent", n)

	return nil
}

func ServeTFTP() error {
	if err := os.MkdirAll("/var/lib/arges/tftp", 0777); err != nil {
		return err
	}

	s := tftp.NewServer(readHandler, nil)

	// A standard TFTP server implementation receives requests on port 69 and
	// allocates a new high port (over 1024) dedicated to that request. In single
	// port mode, the same port is used for transmit and receive. If the server
	// is started on port 69, all communication will be done on port 69.
	// This option is required since the Kubernetes service definition defines a
	// single port.
	s.EnableSinglePort()
	s.SetTimeout(5 * time.Second)

	if err := s.ListenAndServe(":69"); err != nil {
		return err
	}

	return nil
}
