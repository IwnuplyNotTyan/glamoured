package main

import (
	"fmt"
	"os"

	"charm.land/glamour/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <image-url-or-path>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s https://stuff.charm.sh/charm-badge.jpg\n", os.Args[0])
		os.Exit(1)
	}

	imageURL := os.Args[1]

	in := fmt.Sprintf(`# Mosaic Image Rendering

This is an example of rendering an image with Glamour using
[github.com/charmbracelet/x/mosaic](https://github.com/charmbracelet/x/mosaic).

![example](%s)

Pretty cool!
`, imageURL)

	fmt.Println("─── With mosaic (default) ───")
	out, err := glamour.Render(in, "dark")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Print(out)

	fmt.Println("\n─── Without mosaic ───")
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithMosaic(false),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	out, err = r.Render(in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Print(out)
}
