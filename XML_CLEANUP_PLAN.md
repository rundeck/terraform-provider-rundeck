# XML Cleanup Plan

## Current State
- Framework resources (resource_*_framework.go) use JSON-only
- Old SDK resource (resource_job.go) is commented out/disabled  
- Test files still declare `var job JobDetail` but don't use it correctly
- job.go contains 172 XML tags in legacy structs

## Issues Found

### 1. Broken Test Helper
`testAccJobCheckDestroy(&job)` passes a pointer to JobDetail with empty ID.
Function tries to use job.ID but it's always "". Should iterate state instead.

### 2. Unused JobDetail Declarations  
9 tests declare `var job JobDetail` but never populate or use it.

### 3. XML Structs Still Present
- JobDetail (XML struct)
- JobSummary (XML struct)  
- All nested structs with xml tags

## Cleanup Actions

### Step 1: Fix Test Helpers (CRITICAL)
- Remove JobDetail parameter from testAccJobCheckDestroy
- Have it iterate Terraform state to find job resources
- Keep testAccJobCheckExists but simplify

### Step 2: Remove Unused Variables
Remove `var job JobDetail` from all tests that don't use it.

### Step 3: Mark XML Code as DEPRECATED
Add clear comment block at top of job.go:
```
// =============================================================================
// LEGACY XML CODE - DEPRECATED
// =============================================================================  
// The structs below (JobDetail, JobSummary, etc.) use XML tags and are LEGACY.
// They were used by the old SDKv2 resource which is now DISABLED.
//
// **DO NOT USE THESE IN NEW CODE**
//
// For Framework resources, use:
// - JobJSON (for API reads)  
// - jobJSON (lowercase, for API writes)
//
// This code is kept only for:
// 1. Backward compatibility if old SDK resource is re-enabled
// 2. Test helpers that haven't been fully modernized yet
// =============================================================================
```

### Step 4: Document JSON-Only Approach  
Add comment before JobJSON struct explaining it's the modern approach.

### Step 5: Remove encoding/xml import where unused
Check util.go and remove if not needed.

## Files to Modify
1. rundeck/resource_job_test.go - Fix helpers, remove unused vars
2. rundeck/job.go - Add deprecation notices
3. rundeck/util.go - Check if encoding/xml needed
4. rundeck/import_resource_job_test.go - Check for unused JobDetail

## Result
- Clear separation between legacy (XML) and modern (JSON)  
- No confusion for contributors
- Tests work correctly
- All XML marked as deprecated

