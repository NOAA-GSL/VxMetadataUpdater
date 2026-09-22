WITH model_names AS (
    SELECT DISTINCT COALESCE(m.name, m.model) AS model
    FROM {{vxDBTARGET}} AS d UNNEST d.models AS m
    WHERE META(d).id = "MD:matsGui:{{vxDATASET}}:COMMON:V01"
        AND COALESCE(m.name, m.model) IS NOT NULL
        AND COALESCE(m.name, m.model) != ""
)
SELECT RAW mn.model
FROM model_names AS mn LET dd_count = (
        SELECT RAW COUNT(*)
        FROM {{vxDBTARGET}} AS mt
        WHERE mt.type = "DD"
            AND mt.docType = {{vxDOCTYPE}}
            AND mt.subDocType = {{vxSUBDOCTYPE}}
            AND mt.version = "V01"
            AND mt.model = mn.model
    ) [0]
WHERE dd_count > 0
ORDER BY mn.model;