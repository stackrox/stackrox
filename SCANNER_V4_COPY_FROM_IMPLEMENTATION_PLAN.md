# COPY FROM Implementation Plan for Scanner V4 Vulnerability Loading

klape: I didn't have time to read this, but wanted to read it later.  So I have no idea how accurate this is...

## **Overview**

This document outlines a detailed implementation plan to dramatically improve Scanner V4 vulnerability database initialization performance by replacing the current row-by-row INSERT approach with PostgreSQL's high-performance COPY FROM STDIN mechanism.

**Current Performance**: 3-5 minutes for vulnerability loading  
**Target Performance**: 30-60 seconds (5-10x improvement)  
**Stretch Goal**: 15-30 seconds with parallelization (10-20x improvement)

---

## **Phase 1: Analysis & Design (1-2 days)**

### **1.1 ClairCore Database Schema Investigation**

**Action Items:**
```bash
# 1. Examine running Scanner V4 database to understand schema
kubectl exec -it scanner-v4-db-pod -- psql -U postgres -c "\dt"
kubectl exec -it scanner-v4-db-pod -- psql -U postgres -c "\d+ vulnerabilities"

# 2. Identify key tables used by libvuln
kubectl exec -it scanner-v4-db-pod -- psql -U postgres -c "
SELECT table_name FROM information_schema.tables 
WHERE table_schema='public' AND table_name LIKE '%vuln%';"
```

**Expected Tables (based on ClairCore patterns):**
```sql
-- Primary tables (typical ClairCore schema):
vulnerabilities     -- CVE data (id, name, description, severity)
enrichments        -- Additional vulnerability metadata  
update_operations  -- Track update operations and fingerprints
uo_vuln           -- Links vulnerabilities to update operations
```

### **1.2 JSON Bundle Structure Analysis**

**Sample Bundle Investigation:**
```bash
# Download a bundle to examine structure
curl -L "https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip" -o bundles.zip
unzip bundles.zip
zstd -dc oracle.json.zst | head -20
```

**Expected JSON Structure:**
```json
{
  "ref": "uuid-here",
  "updater": "oracle-2024-updater", 
  "fingerprint": "hash",
  "date": "2024-01-01T00:00:00Z",
  "kind": "vulnerability",
  "vulnerability": {
    "id": "CVE-2024-1234",
    "name": "CVE-2024-1234", 
    "description": "...",
    "severity": "HIGH",
    "package": {...},
    "dist": {...},
    "repo": {...}
  }
}
```

---

## **Phase 2: Fast Path Implementation (3-5 days)**

### **2.1 Create Parallel Import Path**

**File**: `/root/src/stackrox/scanner/matcher/updater/vuln/fast_importer.go`

```go
package vuln

import (
    "context"
    "encoding/json"
    "io"
    "github.com/jackc/pgx/v4"
    "github.com/quay/claircore"
    "github.com/quay/claircore/libvuln/driver"
)

// FastImporter provides COPY FROM STDIN based vulnerability import
type FastImporter struct {
    pool *pgx.Conn
}

// VulnRecord represents a flattened vulnerability for COPY
type VulnRecord struct {
    ID           string
    Name         string  
    Description  string
    Severity     string
    Package      string
    DistID       string
    DistName     string
    DistVersion  string
    UpdaterName  string
    Fingerprint  string
    Ref          string
}

func NewFastImporter(pool *pgx.Conn) *FastImporter {
    return &FastImporter{pool: pool}
}

func (f *FastImporter) ImportVulnerabilities(ctx context.Context, r io.Reader) error {
    // Parse JSON stream into VulnRecord structs
    records, err := f.parseJSONStream(r)
    if err != nil {
        return err
    }
    
    // Use PostgreSQL COPY FROM STDIN
    return f.bulkInsert(ctx, records)
}

func (f *FastImporter) parseJSONStream(r io.Reader) ([]VulnRecord, error) {
    var records []VulnRecord
    decoder := json.NewDecoder(r)
    
    for decoder.More() {
        var entry diskEntry
        if err := decoder.Decode(&entry); err != nil {
            return nil, err
        }
        
        // Convert ClairCore vulnerability to flat record
        record := VulnRecord{
            ID:          entry.Vulnerability.ID,
            Name:        entry.Vulnerability.Name,
            Description: entry.Vulnerability.Description,
            Severity:    entry.Vulnerability.NormalizedSeverity.String(),
            // ... map other fields
            UpdaterName: entry.Updater,
            Fingerprint: entry.Fingerprint,
            Ref:         entry.Ref.String(),
        }
        records = append(records, record)
    }
    
    return records, nil
}

func (f *FastImporter) bulkInsert(ctx context.Context, records []VulnRecord) error {
    // Begin transaction for consistency
    tx, err := f.pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)
    
    // Use COPY FROM STDIN for bulk insert
    _, err = tx.CopyFrom(ctx,
        pgx.Identifier{"vulnerabilities"},
        []string{"id", "name", "description", "severity", "package_name", 
                 "dist_id", "dist_name", "dist_version", "updater", "fingerprint", "ref"},
        pgx.CopyFromSlice(len(records), func(i int) ([]interface{}, error) {
            r := records[i]
            return []interface{}{
                r.ID, r.Name, r.Description, r.Severity, r.Package,
                r.DistID, r.DistName, r.DistVersion, r.UpdaterName, 
                r.Fingerprint, r.Ref,
            }, nil
        }))
    
    if err != nil {
        return err
    }
    
    return tx.Commit(ctx)
}
```

### **2.2 Integration Point Modification**

**File**: `/root/src/stackrox/scanner/matcher/updater/vuln/updater.go`

**Modify the `updateBundle` function:**

```go
// Add feature flag for fast import
const useFastImport = true // Eventually make this configurable

func (u *Updater) updateBundle(ctx context.Context, zipF *zip.File, zipTime time.Time, prevTime time.Time) error {
    // ... existing validation logic ...
    
    dec, err := zstd.NewReader(r)
    if err != nil {
        return fmt.Errorf("creating zstd reader: %w", err)
    }
    defer dec.Close()

    // Choose import method based on feature flag
    if useFastImport {
        fastImporter := NewFastImporter(u.pool) // Need access to raw connection
        if err := fastImporter.ImportVulnerabilities(lCtx, dec); err != nil {
            return fmt.Errorf("fast importing vulnerabilities: %w", err)
        }
    } else {
        // Original libvuln import path
        if err := u.importFunc(lCtx, dec); err != nil {
            return fmt.Errorf("importing vulnerabilities: %w", err)
        }
    }
    
    // ... rest of function unchanged ...
}
```

### **2.3 Database Connection Management**

**Challenge**: libvuln uses connection pools, but COPY FROM requires direct connection access.

**Solution**: Extend the postgres store to provide raw connection access:

```go
// In /root/src/stackrox/scanner/datastore/postgres/matcher_store.go

func (s *matcherStore) GetRawConnection(ctx context.Context) (*pgx.Conn, error) {
    return s.pool.Acquire(ctx)
}

func (s *matcherStore) ReleaseConnection(conn *pgx.Conn) {
    s.pool.Release(conn)
}
```

---

## **Phase 3: Schema Compatibility (2-3 days)**

### **3.1 ClairCore Schema Mapping**

**Research Required:**
1. **Examine actual database schema** in running Scanner V4
2. **Map ClairCore JSON fields** to database columns
3. **Handle relational data** (vulnerabilities may reference distributions, packages, etc.)

**Expected Challenges:**
```sql
-- ClairCore likely uses normalized schema:
vulnerabilities (id, name, description, severity, ...)
packages (id, name, version, ...)  
distributions (id, name, version, ...)
uo_vulns (vulnerability_id, update_operation_id, ...)

-- Our COPY approach needs to handle these relationships
```

### **3.2 Data Transformation Layer**

**Create schema-aware transformer:**

```go
type SchemaMapper struct {
    packageCache    map[string]int64    // package_name -> package_id
    distCache       map[string]int64    // dist_key -> dist_id  
    vulnInsertQueue []VulnerabilityRow
    pkgInsertQueue  []PackageRow
    distInsertQueue []DistributionRow
}

func (m *SchemaMapper) ProcessVulnerability(vuln *claircore.Vulnerability) error {
    // 1. Ensure package exists, get/create ID
    pkgID := m.ensurePackage(vuln.Package)
    
    // 2. Ensure distribution exists, get/create ID  
    distID := m.ensureDistribution(vuln.Dist)
    
    // 3. Queue vulnerability insert with foreign keys
    m.vulnInsertQueue = append(m.vulnInsertQueue, VulnerabilityRow{
        ID:             vuln.ID,
        Name:           vuln.Name,
        Description:    vuln.Description, 
        Severity:       vuln.NormalizedSeverity.String(),
        PackageID:      pkgID,
        DistributionID: distID,
    })
    
    return nil
}

func (m *SchemaMapper) FlushToDatabase(ctx context.Context, conn *pgx.Conn) error {
    // 1. COPY packages first (due to foreign key constraints)
    if err := m.insertPackages(ctx, conn); err != nil {
        return err
    }
    
    // 2. COPY distributions  
    if err := m.insertDistributions(ctx, conn); err != nil {
        return err
    }
    
    // 3. COPY vulnerabilities
    if err := m.insertVulnerabilities(ctx, conn); err != nil {
        return err
    }
    
    return nil
}
```

---

## **Phase 4: Performance Optimization (1-2 days)**

### **4.1 Batch Processing**

```go
const (
    defaultBatchSize = 10000
    maxBatchSize     = 50000
)

func (f *FastImporter) ImportVulnerabilities(ctx context.Context, r io.Reader) error {
    batchProcessor := NewBatchProcessor(f.pool, defaultBatchSize)
    
    return parseJSONStreamBatched(r, func(batch []VulnRecord) error {
        return batchProcessor.ProcessBatch(ctx, batch)
    })
}

type BatchProcessor struct {
    pool      *pgx.Conn
    batchSize int
}

func (b *BatchProcessor) ProcessBatch(ctx context.Context, records []VulnRecord) error {
    // Split large batches if needed
    for i := 0; i < len(records); i += b.batchSize {
        end := i + b.batchSize
        if end > len(records) {
            end = len(records)
        }
        
        if err := b.copyBatch(ctx, records[i:end]); err != nil {
            return err
        }
    }
    return nil
}
```

### **4.2 Parallel Bundle Processing**

```go
// Modify the main update loop to process bundles in parallel
func (u *Updater) updateBundles(ctx context.Context, bundles []*zip.File, zipTime time.Time, prevTime time.Time) error {
    const maxConcurrency = 3 // Limit to avoid overwhelming database
    
    sem := make(chan struct{}, maxConcurrency)
    errCh := make(chan error, len(bundles))
    
    for _, bundle := range bundles {
        go func(b *zip.File) {
            sem <- struct{}{}
            defer func() { <-sem }()
            
            errCh <- u.updateBundle(ctx, b, zipTime, prevTime)
        }(bundle)
    }
    
    // Wait for all bundles and collect errors
    var errors []error
    for i := 0; i < len(bundles); i++ {
        if err := <-errCh; err != nil {
            errors = append(errors, err)
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("bundle processing errors: %v", errors)
    }
    
    return nil
}
```

---

## **Phase 5: Configuration & Rollback (1 day)**

### **5.1 Feature Flag Implementation**

```go
// In config.go
type Config struct {
    // ... existing fields ...
    UseFastVulnImport bool `yaml:"use_fast_vuln_import"`
}

// Environment variable support
func (c *Config) loadFromEnv() {
    if val := os.Getenv("SCANNER_V4_FAST_IMPORT"); val == "true" {
        c.UseFastVulnImport = true
    }
}
```

### **5.2 Rollback Strategy**

```go
func (u *Updater) updateBundle(ctx context.Context, zipF *zip.File, zipTime time.Time, prevTime time.Time) error {
    // ... existing validation ...
    
    var importErr error
    if u.config.UseFastVulnImport {
        zlog.Info(ctx).Msg("using fast COPY FROM import")
        importErr = u.fastImportFunc(lCtx, dec)
    } else {
        zlog.Info(ctx).Msg("using standard libvuln import")  
        importErr = u.importFunc(lCtx, dec)
    }
    
    if importErr != nil {
        if u.config.UseFastVulnImport {
            // Fallback to standard import on error
            zlog.Warn(ctx).Err(importErr).Msg("fast import failed, falling back to standard import")
            
            // Re-open bundle stream
            r2, err := zipF.Open()
            if err != nil {
                return err
            }
            defer r2.Close()
            
            dec2, err := zstd.NewReader(r2)
            if err != nil {
                return err
            }
            defer dec2.Close()
            
            importErr = u.importFunc(lCtx, dec2)
        }
        
        if importErr != nil {
            return fmt.Errorf("importing vulnerabilities: %w", importErr)
        }
    }
    
    // ... rest unchanged ...
}
```

---

## **Phase 6: Testing & Validation (2-3 days)**

### **6.1 Unit Tests**

```go
func TestFastImporter_ImportVulnerabilities(t *testing.T) {
    // Test with sample JSON vulnerability data
    testData := `{"ref":"uuid","updater":"test","vulnerability":{"id":"CVE-2024-1","name":"test-vuln"}}`
    
    importer := NewFastImporter(testDB)
    err := importer.ImportVulnerabilities(ctx, strings.NewReader(testData))
    
    assert.NoError(t, err)
    
    // Verify data was inserted correctly
    var count int
    err = testDB.QueryRow(ctx, "SELECT COUNT(*) FROM vulnerabilities WHERE id = 'CVE-2024-1'").Scan(&count)
    assert.NoError(t, err)
    assert.Equal(t, 1, count)
}
```

### **6.2 Performance Benchmarks**

```go
func BenchmarkImportMethods(b *testing.B) {
    testBundle := generateTestBundle(10000) // 10k vulnerabilities
    
    b.Run("StandardImport", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            updater.importFunc(ctx, testBundle)
        }
    })
    
    b.Run("FastImport", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            fastImporter.ImportVulnerabilities(ctx, testBundle)
        }
    })
}
```

### **6.3 Integration Testing**

```go
func TestE2E_FastImportConsistency(t *testing.T) {
    // Import same bundle with both methods
    standardResult := importWithStandardMethod(testBundle)
    fastResult := importWithFastMethod(testBundle)
    
    // Verify identical results
    assert.Equal(t, standardResult.VulnCount, fastResult.VulnCount)
    assert.Equal(t, standardResult.PackageCount, fastResult.PackageCount)
    
    // Verify vulnerability matching still works
    report := scanner.ScanImage(ctx, testImage)
    assert.True(t, len(report.Vulnerabilities) > 0)
}
```

---

## **Phase 7: Deployment Strategy (1 day)**

### **7.1 Gradual Rollout**

```yaml
# Development deployment with fast import
env:
- name: SCANNER_V4_FAST_IMPORT
  value: "true"

# Production deployment (initially disabled)  
env:
- name: SCANNER_V4_FAST_IMPORT
  value: "false"
```

### **7.2 Monitoring & Metrics**

```go
// Add metrics to track performance
func (f *FastImporter) ImportVulnerabilities(ctx context.Context, r io.Reader) error {
    start := time.Now()
    defer func() {
        importDuration.Observe(time.Since(start).Seconds())
    }()
    
    recordCount := 0
    // ... import logic ...
    
    importedVulnerabilities.Add(float64(recordCount))
    return nil
}
```

---

## **Expected Performance Improvements**

| Method | Time | Improvement |
|--------|------|-------------|
| **Current (libvuln)** | 3-5 minutes | Baseline |
| **COPY FROM STDIN** | 30-60 seconds | **5-10x faster** |
| **COPY + Parallel** | 15-30 seconds | **10-20x faster** |
| **Pre-built DB** | 10-20 seconds | **15-30x faster** |

---

## **Risk Mitigation**

1. **Feature Flag**: Easy disable if issues arise
2. **Fallback Path**: Automatic retry with standard method
3. **Schema Validation**: Extensive testing against ClairCore compatibility
4. **Gradual Rollout**: Test in development before production
5. **Monitoring**: Track performance and error rates

---

## **Alternative Approaches Considered**

### **A. Pre-built Database Images**
**Concept**: Build Scanner V4 DB images with vulnerability data already loaded

**Advantages**:
- **Instant startup** (no loading required)
- **Consistent data** across deployments
- **No network dependencies**

**Disadvantages**:
- **Large image sizes** (GB+ with full vulnerability data)
- **Update complexity** (requires rebuilding images)
- **Storage overhead**

### **B. Bundle Pre-loading (Already Available)**
**Current Status**: Feature exists but `/db-init.dump.zst` file is missing
**Implementation**: Create database dumps with pre-loaded vulnerability data
**Expected Performance**: 10-20 seconds initialization

### **C. PostgreSQL Optimization Techniques**

```postgresql
# Optimize for bulk loading
wal_level = minimal                    # Reduce WAL overhead
max_wal_senders = 0                   # Disable replication
checkpoint_segments = 64              # Reduce checkpoint frequency
checkpoint_completion_target = 0.9    # Spread checkpoints
shared_buffers = 4GB                  # More memory for buffers
work_mem = 256MB                      # Larger sort/hash memory
maintenance_work_mem = 2GB            # For CREATE INDEX operations
fsync = off                           # Disable for initial load (risky)
```

---

## **Timeline Summary**

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| **Phase 1** | 1-2 days | Schema analysis, JSON structure understanding |
| **Phase 2** | 3-5 days | Working COPY FROM implementation |
| **Phase 3** | 2-3 days | Schema compatibility layer |
| **Phase 4** | 1-2 days | Performance optimizations |
| **Phase 5** | 1 day | Configuration and rollback mechanisms |
| **Phase 6** | 2-3 days | Comprehensive testing |
| **Phase 7** | 1 day | Deployment strategy |
| **Total** | **11-17 days** | Production-ready implementation |

---

## **Success Metrics**

1. **Performance**: 5-10x faster vulnerability loading (target: < 60 seconds)
2. **Reliability**: Zero data corruption, full compatibility with existing scans
3. **Monitoring**: Clear metrics on import performance and error rates
4. **Rollback**: Seamless fallback to standard import if issues occur
5. **Maintainability**: Clean code integration with existing ClairCore architecture

This implementation plan provides a comprehensive path to dramatically improve Scanner V4 initialization performance while maintaining full compatibility with the existing ClairCore ecosystem.
