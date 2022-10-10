package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ItsNotGoodName/x-ipc-viewer/mosaic"
)

type Layout string

func (c Layout) IsAuto() bool {
	return c == "" || c == "auto"
}

func (c Layout) IsManual() bool {
	return c == "manual"
}

type LayoutManual struct {
	X string
	Y string
	W string
	H string
}

func parseRatio(ratio string) (float32, error) {
	if num, err := strconv.ParseFloat(ratio, 32); err == nil {
		return float32(num), err
	}

	f := strings.Split(ratio, "/")
	fLen := len(f)
	if fLen == 2 {
		num, err := strconv.ParseFloat(f[0], 32)
		if err != nil {
			return 0, err
		}

		den, err := strconv.ParseFloat(f[1], 32)
		if err != nil {
			return 0, err
		}

		return float32(num) / float32(den), nil
	}

	return 0, fmt.Errorf("%s: invalid float", ratio)
}

func parseLayoutManualWindow(lmr LayoutManual) (mosaic.LayoutManualWindow, error) {
	x, err := parseRatio(lmr.X)
	if err != nil {
		return mosaic.LayoutManualWindow{}, fmt.Errorf("X=%w", err)
	}

	y, err := parseRatio(lmr.Y)
	if err != nil {
		return mosaic.LayoutManualWindow{}, fmt.Errorf("Y=%w", err)
	}

	w, err := parseRatio(lmr.W)
	if err != nil {
		return mosaic.LayoutManualWindow{}, fmt.Errorf("W=%w", err)
	}

	h, err := parseRatio(lmr.H)
	if err != nil {
		return mosaic.LayoutManualWindow{}, fmt.Errorf("H=%w", err)
	}

	return mosaic.LayoutManualWindow{
		X: x,
		Y: y,
		W: w,
		H: h,
	}, nil
}
