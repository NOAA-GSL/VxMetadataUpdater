package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// safeIdentRe restricts query parameter substitution to prevent SQL injection.
var safeIdentRe = regexp.MustCompile(`^[a-zA-Z0-9_.:\-/]+$`)

// validateQueryParam fatals if value contains characters outside [a-zA-Z0-9_.:\-/],
// preventing SQL injection via template substitution.
func validateQueryParam(name, value string) {
	if value != "" && !safeIdentRe.MatchString(value) {
		log.Fatalf("query parameter %q contains invalid characters: %q", name, value)
	}
}

// readSQLTemplate searches for name in sqls/ relative to the working directory,
// the executable directory, and the source file directory (in that order).
func readSQLTemplate(name string) ([]byte, error) {
	candidates := []string{filepath.Join("sqls", name)}

	if exePath, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exePath), "sqls", name))
	}
	if _, sourcePath, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(sourcePath), "sqls", name))
	}

	var lastErr error
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}

		fileContent, err := os.ReadFile(candidate)
		if err == nil {
			return fileContent, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("unable to read SQL template %q: %w", name, lastErr)
}

func getModels(conn CbConnection, dataset string, app string, doctype string, subDocType string) (jsonOut []string) {
	log.Println("getModels(" + dataset + "," + app + "," + doctype + "," + subDocType + ")")
	validateQueryParam("doctype", doctype)
	validateQueryParam("subDocType", subDocType)
	fileContent, err := readSQLTemplate("getModels.sql")
	if err != nil {
		log.Fatal(err)
	}
	tmplGetModelsSQL := string(fileContent)
	tmplGetModelsSQL = strings.Replace(tmplGetModelsSQL, "{{vxDBTARGET}}", conn.vxDBTARGET, -1)
	tmplGetModelsSQL = strings.Replace(tmplGetModelsSQL, "{{vxDOCTYPE}}", doctype, -1)
	tmplGetModelsSQL = strings.Replace(tmplGetModelsSQL, "{{vxSUBDOCTYPE}}", subDocType, -1)

	models_requiring_metadata := queryWithSQLStringSA(conn.Scope, tmplGetModelsSQL)
	return models_requiring_metadata
}

// getModelsNoData returns model names that have a metadata document but no matching DD data records.
func getModelsNoData(conn CbConnection, dataset string, app string, doctype string, subDocType string) (jsonOut []string) {
	log.Println("getModelsNoData(" + dataset + "," + app + "," + doctype + "," + subDocType + ")")
	validateQueryParam("app", app)
	validateQueryParam("doctype", doctype)
	validateQueryParam("subDocType", subDocType)
	fileContent, err := readSQLTemplate("getModelsNoData.sql")
	if err != nil {
		log.Fatal(err)
	}
	tmplgetModelsNoDataSQL := string(fileContent)
	tmplgetModelsNoDataSQL = strings.Replace(tmplgetModelsNoDataSQL, "{{vxDBTARGET}}", conn.vxDBTARGET, -1)
	tmplgetModelsNoDataSQL = strings.Replace(tmplgetModelsNoDataSQL, "{{vxDOCTYPE}}", doctype, -1)
	tmplgetModelsNoDataSQL = strings.Replace(tmplgetModelsNoDataSQL, "{{vxAPP}}", app, -1)
	tmplgetModelsNoDataSQL = strings.Replace(tmplgetModelsNoDataSQL, "{{vxSUBDOCTYPE}}", subDocType, -1)
	models_with_metatada_but_no_data := queryWithSQLStringSA(conn.Scope, tmplgetModelsNoDataSQL)
	return models_with_metatada_but_no_data
}

func getDistinctDataKeys(conn CbConnection, dataset string, app string, doctype string, subDocType string, model string) (rv []string) {
	log.Println("getDistinctDataKeys(" + dataset + "," + app + "," + doctype + "," + subDocType + "," + model + ")")
	validateQueryParam("doctype", doctype)
	validateQueryParam("subDocType", subDocType)
	validateQueryParam("model", model)
	fileContent, err := readSQLTemplate("getDistinctDataKeys.sql")
	if err != nil {
		log.Fatal(err)
	}
	tmplSQL := string(fileContent)
	tmplSQL = strings.Replace(tmplSQL, "{{vxDBTARGET}}", conn.vxDBTARGET, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxDOCTYPE}}", doctype, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxSUBDOCTYPE}}", subDocType, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxMODEL}}", model, -1)
	log.Println(tmplSQL)
	return queryWithSQLStringSA(conn.Scope, tmplSQL)
}

func getDistinctFcstLen(conn CbConnection, dataset string, app string, doctype string, subDocType string, model string) (rv []int) {
	log.Println("getDistinctFcstLen(" + dataset + "," + app + "," + doctype + "," + subDocType + "," + model + ")")
	validateQueryParam("doctype", doctype)
	validateQueryParam("subDocType", subDocType)
	validateQueryParam("model", model)
	fileContent, err := readSQLTemplate("getDistinctFcstLen.sql")
	if err != nil {
		log.Fatal(err)
	}
	tmplSQL := string(fileContent)
	tmplSQL = strings.Replace(tmplSQL, "{{vxDBTARGET}}", conn.vxDBTARGET, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxDOCTYPE}}", doctype, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxSUBDOCTYPE}}", subDocType, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxMODEL}}", model, -1)
	result := queryWithSQLStringIA(conn.Scope, tmplSQL)
	return result
}

func getDistinctRegion(conn CbConnection, dataset string, app string, doctype string, subDocType string, model string) (rv []string) {
	log.Println("getDistinctRegion(" + dataset + "," + app + "," + doctype + "," + subDocType + "," + model + ")")
	validateQueryParam("doctype", doctype)
	validateQueryParam("subDocType", subDocType)
	validateQueryParam("model", model)
	fileContent, err := readSQLTemplate("getDistinctRegion.sql")
	if err != nil {
		log.Fatal(err)
	}
	tmplSQL := string(fileContent)
	tmplSQL = strings.Replace(tmplSQL, "{{vxDBTARGET}}", conn.vxDBTARGET, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxDOCTYPE}}", doctype, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxSUBDOCTYPE}}", subDocType, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxMODEL}}", model, -1)
	result := queryWithSQLStringSA(conn.Scope, tmplSQL)
	return result
}

func getDistinctDisplayText(conn CbConnection, dataset string, app string, doctype string, subDocType string, model string) (rv []string) {
	log.Println("getDistinctDisplayText(" + dataset + "," + app + "," + doctype + "," + subDocType + "," + model + ")")
	validateQueryParam("doctype", doctype)
	validateQueryParam("subDocType", subDocType)
	validateQueryParam("model", model)
	fileContent, err := readSQLTemplate("getDistinctDisplayText.sql")
	if err != nil {
		log.Fatal(err)
	}
	tmplSQL := string(fileContent)
	tmplSQL = strings.Replace(tmplSQL, "{{vxDBTARGET}}", conn.vxDBTARGET, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxDOCTYPE}}", doctype, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxSUBDOCTYPE}}", subDocType, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxMODEL}}", model, -1)
	result := queryWithSQLStringSA(conn.Scope, tmplSQL)
	return result
}

func getDistinctDisplayCategory(conn CbConnection, dataset string, app string, doctype string, subDocType string, model string) (rv []int) {
	log.Println("getDistinctDisplayCategory(" + dataset + "," + app + "," + doctype + "," + subDocType + "," + model + ")")
	validateQueryParam("doctype", doctype)
	validateQueryParam("subDocType", subDocType)
	validateQueryParam("model", model)
	fileContent, err := readSQLTemplate("getDistinctDisplayCategory.sql")
	if err != nil {
		log.Fatal(err)
	}
	tmplSQL := string(fileContent)
	tmplSQL = strings.Replace(tmplSQL, "{{vxDBTARGET}}", conn.vxDBTARGET, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxDOCTYPE}}", doctype, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxSUBDOCTYPE}}", subDocType, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxMODEL}}", model, -1)
	result := queryWithSQLStringIA(conn.Scope, tmplSQL)
	return result
}

func getDistinctDisplayOrder(conn CbConnection, dataset string, app string, doctype string, subDocType string, model string, mindx int) (rv []int) {
	log.Println("getDistinctDisplayOrder(" + dataset + "," + app + "," + doctype + "," + subDocType + "," + model + ")")
	validateQueryParam("doctype", doctype)
	validateQueryParam("subDocType", subDocType)
	validateQueryParam("model", model)
	fileContent, err := readSQLTemplate("getDistinctDisplayOrder.sql")
	if err != nil {
		log.Fatal(err)
	}
	tmplSQL := string(fileContent)
	tmplSQL = strings.Replace(tmplSQL, "{{vxDBTARGET}}", conn.vxDBTARGET, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxDOCTYPE}}", doctype, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxSUBDOCTYPE}}", subDocType, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxMODEL}}", model, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{mindx}}", strconv.Itoa(mindx), -1)
	result := queryWithSQLStringIA(conn.Scope, tmplSQL)
	return result
}

func getMinMaxCountFloor(conn CbConnection, dataset string, app string, doctype string, subDocType string, model string) (jsonOut []interface{}) {
	log.Println("getMinMaxCountFloor(" + dataset + "," + app + "," + doctype + "," + subDocType + "," + model + ")")
	validateQueryParam("doctype", doctype)
	validateQueryParam("subDocType", subDocType)
	validateQueryParam("model", model)
	fileContent, err := readSQLTemplate("getMinMaxCountFloor.sql")
	if err != nil {
		log.Fatal(err)
	}
	tmplSQL := string(fileContent)
	tmplSQL = strings.Replace(tmplSQL, "{{vxDBTARGET}}", conn.vxDBTARGET, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxDOCTYPE}}", doctype, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxSUBDOCTYPE}}", subDocType, -1)
	tmplSQL = strings.Replace(tmplSQL, "{{vxMODEL}}", model, -1)
	result := queryWithSQLStringMAP(conn.Scope, tmplSQL)
	return result
}
