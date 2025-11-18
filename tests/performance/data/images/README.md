# Test Images

This directory is for storing test images used in performance tests.

## Usage

Place sample images here for testing:

```bash
# Copy test images
cp /path/to/test/images/*.jpg tests/performance/data/images/

# Or create test images with ImageMagick
convert -size 1920x1080 xc:white test-image-1.jpg
convert -size 1920x1080 gradient:blue-red test-image-2.jpg
```

## Note

The k6 tests generate minimal test JPEGs programmatically, so this directory is optional for basic testing. However, for more realistic performance testing, you can place actual image files here and modify the k6 scripts to use them.

## Git

Test images are git-ignored to keep the repository size small. Each developer should generate or copy their own test images.
