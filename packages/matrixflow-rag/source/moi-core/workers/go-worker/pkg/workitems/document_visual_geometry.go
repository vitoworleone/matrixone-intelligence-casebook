package workitems

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"path"
	"strings"

	moi "github.com/matrixflow/moi-core/go-sdk"
)

func documentVisualImageBBoxToPageBBox(imageBBox []float64, pageWidth, pageHeight float64, imageWidth, imageHeight int) ([]float64, error) {
	if len(imageBBox) != 4 {
		return nil, fmt.Errorf("image bbox must have 4 coordinates")
	}
	if pageWidth <= 0 || pageHeight <= 0 || imageWidth <= 0 || imageHeight <= 0 {
		return nil, fmt.Errorf("page and rendered image dimensions are required")
	}
	if err := validateDocumentVisualImageBBox(imageBBox, imageWidth, imageHeight, "image bbox"); err != nil {
		return nil, err
	}
	if documentVisualShouldRotateBBox(pageWidth, pageHeight, imageWidth, imageHeight) {
		return []float64{
			imageBBox[1] / float64(imageHeight) * pageWidth,
			imageBBox[0] / float64(imageWidth) * pageHeight,
			imageBBox[3] / float64(imageHeight) * pageWidth,
			imageBBox[2] / float64(imageWidth) * pageHeight,
		}, nil
	}
	return []float64{
		imageBBox[0] / float64(imageWidth) * pageWidth,
		imageBBox[1] / float64(imageHeight) * pageHeight,
		imageBBox[2] / float64(imageWidth) * pageWidth,
		imageBBox[3] / float64(imageHeight) * pageHeight,
	}, nil
}

func cropDocumentVisualPageImage(pageImageBytes []byte, bbox []float64, pageWidth, pageHeight float64, imageWidth, imageHeight int, objectID string) ([]byte, error) {
	if err := validateDocumentVisualPageBBox(bbox, pageWidth, pageHeight, fmt.Sprintf("object %s bbox", objectID)); err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(pageImageBytes))
	if err != nil {
		return nil, fmt.Errorf("document_visual.parse: decode page image for object %s: %w", objectID, err)
	}
	x0, y0, x1, y1 := documentVisualCropRect(bbox, pageWidth, pageHeight, imageWidth, imageHeight)
	dst := image.NewRGBA(image.Rect(0, 0, x1-x0, y1-y0))
	draw.Draw(dst, dst.Bounds(), img, image.Point{X: x0, Y: y0}, draw.Src)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 92}); err != nil {
		return nil, fmt.Errorf("document_visual.parse: encode object image %s: %w", objectID, err)
	}
	return buf.Bytes(), nil
}

func validateDocumentVisualImageBBox(bbox []float64, imageWidth, imageHeight int, label string) error {
	if len(bbox) != 4 {
		return fmt.Errorf("document_visual.parse: %s must have 4 coordinates", label)
	}
	if bbox[0] < 0 || bbox[1] < 0 || bbox[2] > float64(imageWidth) || bbox[3] > float64(imageHeight) {
		return fmt.Errorf("document_visual.parse: %s out of image bounds: bbox=%v image=%dx%d", label, bbox, imageWidth, imageHeight)
	}
	if bbox[2] <= bbox[0] || bbox[3] <= bbox[1] {
		return fmt.Errorf("document_visual.parse: %s is invalid: bbox=%v", label, bbox)
	}
	return nil
}

func validateDocumentVisualPageBBox(bbox []float64, pageWidth, pageHeight float64, label string) error {
	if len(bbox) != 4 {
		return fmt.Errorf("document_visual.parse: %s must have 4 coordinates", label)
	}
	if pageWidth <= 0 || pageHeight <= 0 {
		return fmt.Errorf("document_visual.parse: page dimensions are required for %s", label)
	}
	if bbox[0] < 0 || bbox[1] < 0 || bbox[2] > pageWidth || bbox[3] > pageHeight {
		return fmt.Errorf("document_visual.parse: %s out of page bounds: bbox=%v page=%vx%v", label, bbox, pageWidth, pageHeight)
	}
	if bbox[2] <= bbox[0] || bbox[3] <= bbox[1] {
		return fmt.Errorf("document_visual.parse: %s is invalid: bbox=%v", label, bbox)
	}
	return nil
}

func cropAndUploadDocumentVisualObject(ctx context.Context, client *moi.Client, workspaceID, sourceFileName string, obj documentVisualObject, asset documentVisualPageAsset, pageWidth, pageHeight float64) (string, error) {
	if len(asset.Bytes) == 0 {
		return "", fmt.Errorf("document_visual.parse: cannot crop object %s without page image bytes", obj.ObjectID)
	}
	if pageWidth <= 0 || pageHeight <= 0 || asset.Width <= 0 || asset.Height <= 0 {
		return "", fmt.Errorf("document_visual.parse: cannot crop object %s without page dimensions", obj.ObjectID)
	}
	imageBytes, err := cropDocumentVisualPageImage(asset.Bytes, obj.BBox, pageWidth, pageHeight, asset.Width, asset.Height, obj.ObjectID)
	if err != nil {
		return "", err
	}
	resp, err := client.Files().UploadBytes(ctx, workspaceID, documentVisualObjectImageName(sourceFileName, obj), imageBytes)
	if err != nil {
		return "", fmt.Errorf("document_visual.parse: upload object image %s: %w", obj.ObjectID, err)
	}
	return resp.FileID, nil
}

func documentVisualCropRect(bbox []float64, pageWidth, pageHeight float64, imageWidth, imageHeight int) (int, int, int, int) {
	scaleWidth := pageWidth
	scaleHeight := pageHeight
	box := append([]float64(nil), bbox...)
	if documentVisualShouldRotateBBox(pageWidth, pageHeight, imageWidth, imageHeight) && len(box) == 4 {
		scaleWidth, scaleHeight = pageHeight, pageWidth
		box = []float64{bbox[1], bbox[0], bbox[3], bbox[2]}
	}
	x0 := int(clampFloat(box[0]/scaleWidth*float64(imageWidth), 0, float64(imageWidth-1)))
	y0 := int(clampFloat(box[1]/scaleHeight*float64(imageHeight), 0, float64(imageHeight-1)))
	x1 := int(clampFloat(box[2]/scaleWidth*float64(imageWidth), float64(x0+1), float64(imageWidth)))
	y1 := int(clampFloat(box[3]/scaleHeight*float64(imageHeight), float64(y0+1), float64(imageHeight)))
	return x0, y0, x1, y1
}

func documentVisualShouldRotateBBox(pageWidth, pageHeight float64, imageWidth, imageHeight int) bool {
	if pageWidth <= 0 || pageHeight <= 0 || imageWidth <= 0 || imageHeight <= 0 {
		return false
	}
	pageLandscape := pageWidth >= pageHeight
	imageLandscape := imageWidth >= imageHeight
	return pageLandscape != imageLandscape
}

func isDocumentVisualImageSource(source documentVisualSource, raw []byte) bool {
	mime := strings.ToLower(source.MimeType)
	ext := strings.ToLower(path.Ext(source.FileName))
	if strings.HasPrefix(mime, "image/") || ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".webp" {
		return true
	}
	_, _, err := image.DecodeConfig(bytes.NewReader(raw))
	return err == nil
}

func imageSize(data []byte) (int, int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}
