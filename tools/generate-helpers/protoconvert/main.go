package main

import (
	"fmt"
	"io"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	_ "github.com/stackrox/rox/generated/api/v1"
	_ "github.com/stackrox/rox/generated/storage"
	_ "github.com/stackrox/rox/generated/test"
	_ "github.com/stackrox/rox/generated/test2"
	"github.com/stackrox/rox/pkg/utils"
)

const header = `package protoconvert

import (
	"%s"
	"%s"
)

`

func main() {
	c := &cobra.Command{
		Use: "generate protobuf conversions",
	}

	var fromStr, fromPath, toStr, toPath, file string
	var bidirectional bool
	c.Flags().StringVar(&fromStr, "from", "", "the proto to convert from")
	utils.Must(c.MarkFlagRequired("from"))
	c.Flags().StringVar(&fromPath, "from-path", "github.com/stackrox/rox/generated/storage", "package path of the proto to convert from")

	c.Flags().StringVar(&toStr, "to", "", "the proto to convert to")
	utils.Must(c.MarkFlagRequired("to"))
	c.Flags().StringVar(&toPath, "to-path", "github.com/stackrox/rox/generated/api/v1", "package path of the proto to convert to")

	c.Flags().BoolVar(&bidirectional, "bidirectional", false, "specifies that you want the conversion in both ways (e.g. from->to and to->from)")

	c.Flags().StringVar(&file, "file", "", "output filename")
	utils.Must(c.MarkFlagRequired("file"))

	c.RunE = func(*cobra.Command, []string) error {
		from := proto.MessageType(fromStr)
		if from == nil {
			return errors.Errorf("could not find message for type: %s", from)
		}
		to := proto.MessageType(toStr)
		if to == nil {
			return errors.Errorf("could not find message for type: %s", to)
		}

		if err := os.Remove(file); err != nil {
			return err
		}
		f, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR, 0700)
		if err != nil {
			return err
		}

		setupFile(f, fromPath, toPath)

		walk(f, from, to)
		if bidirectional {
			walk(f, to, from)
		}
		return f.Close()
	}
	if err := c.Execute(); err != nil {
		panic(err)
	}
}

func setupFile(w io.Writer, p1Path, p2Path string) {
	if p1Path > p2Path {
		p1Path, p2Path = p2Path, p1Path
	}
	if _, err := w.Write([]byte(fmt.Sprintf(header, p1Path, p2Path))); err != nil {
		panic(err)
	}
}
