# RabbitMQ Contract Tests

This directory contains contract tests for RabbitMQ messages exchanged between services in the photo-tags system.

## Overview

Contract tests ensure that all services agree on the structure and format of messages they exchange. This prevents integration issues when services are developed and deployed independently.

## Message Types

The following message types are validated:

### 1. ImageUpload
- **Queue**: `image_upload`
- **Producer**: Gateway Service
- **Consumer**: Analyzer Service
- **Purpose**: Notifies about new image uploads

### 2. MetadataGenerated
- **Queue**: `metadata_generated`
- **Producer**: Analyzer Service
- **Consumer**: Processor Service
- **Purpose**: Contains AI-generated image metadata

### 3. ImageProcessed
- **Queue**: `image_processed`
- **Producer**: Processor Service
- **Consumer**: Gateway Service
- **Purpose**: Reports image processing completion or failure

## Test Structure

### Schema Validation Tests (`schema_validation_test.go`)
- Validates messages against JSON Schema definitions
- Ensures required fields are present
- Verifies data types and formats
- Checks for additional properties

### Serialization Tests (`serialization_test.go`)
- Tests JSON marshaling and unmarshaling
- Verifies field names match the contract
- Ensures cross-service compatibility
- Tests special cases (empty arrays, omitted fields)

### Compatibility Tests (`compatibility_test.go`)
- **Backwards Compatibility**: Ensures current code can read old message formats
- **Forward Compatibility**: Tests handling of future message additions
- **Message Evolution**: Validates schema changes over time
- **Real-world Scenarios**: Tests gradual rollout and mixed version scenarios

## JSON Schemas

All JSON schemas are located in the `schemas/` directory:

- `metadata.json` - Metadata structure
- `image_upload.json` - ImageUpload message schema
- `metadata_generated.json` - MetadataGenerated message schema
- `image_processed.json` - ImageProcessed message schema

Schemas follow JSON Schema Draft 07 specification.

## Running Tests

### Run all contract tests
```bash
cd tests/contracts
go test -v ./...
```

### Run specific test suite
```bash
go test -v -run TestImageUploadSchema
go test -v -run TestSerialization
go test -v -run TestBackwardsCompatibility
```

### Run from project root
```bash
make test  # Runs all tests including contracts
```

## Adding New Message Types

When adding a new message type:

1. **Define the Go struct** in `pkg/models/messages.go`
2. **Create JSON schema** in `tests/contracts/schemas/`
3. **Add validation tests** in `schema_validation_test.go`
4. **Add serialization tests** in `serialization_test.go`
5. **Add compatibility tests** in `compatibility_test.go`

## Best Practices

### Schema Changes

#### Safe Changes (Backwards Compatible)
- ✅ Adding new optional fields
- ✅ Making required fields optional
- ✅ Relaxing validation constraints
- ✅ Adding new enum values (if code handles unknown values)

#### Breaking Changes (Avoid or Plan Carefully)
- ❌ Removing fields
- ❌ Renaming fields
- ❌ Changing field types
- ❌ Making optional fields required
- ❌ Removing enum values
- ❌ Adding stricter validation

### Deployment Strategy

When making breaking changes:

1. **Phase 1**: Add new fields as optional, keep old fields
2. **Phase 2**: Update all consumers to use new fields
3. **Phase 3**: Update all producers to send new fields
4. **Phase 4**: Verify all services are updated
5. **Phase 5**: Remove old fields and mark as breaking change

### Testing During Development

Run contract tests frequently during development:

```bash
# Watch mode (requires entr or similar)
find . -name "*.go" | entr -c go test -v ./...

# Quick test during development
go test -v -short ./...
```

## Troubleshooting

### Schema Validation Failures

If schema validation fails:
1. Check the JSON schema file for correctness
2. Verify the Go struct tags match schema field names
3. Ensure required fields are present in test data
4. Check data types and formats

### Serialization Issues

If serialization tests fail:
1. Verify JSON tags on Go structs
2. Check for missing `omitempty` tags
3. Ensure time.Time fields use RFC3339 format
4. Verify nested structures serialize correctly

### Compatibility Issues

If compatibility tests fail:
1. Review recent schema changes
2. Check if breaking changes were introduced
3. Verify all services are using compatible versions
4. Test with real messages from production (sanitized)

## Integration with CI/CD

Contract tests are automatically run as part of:
- Pre-commit hooks
- Pull request checks
- CI/CD pipeline

All contract tests must pass before merging changes.

## Dependencies

- `github.com/stretchr/testify` - Testing assertions
- `github.com/xeipuuv/gojsonschema` - JSON Schema validation

## References

- [JSON Schema Specification](https://json-schema.org/)
- [Contract Testing Guide](https://martinfowler.com/bliki/ContractTest.html)
- [RabbitMQ Best Practices](https://www.rabbitmq.com/tutorials/tutorial-one-go.html)
