WITH model_names AS (
    SELECT DISTINCT RAW COALESCE(m.name, m.model)
    FROM {{vxDBTARGET}} AS d UNNEST d.models AS m
    WHERE META(d).id = "MD:matsGui:{{vxDATASET}}:COMMON:V01"
        AND COALESCE(m.name, m.model) IS NOT NULL
        AND COALESCE(m.name, m.model) != ""
)
SELECT RAW mn
FROM model_names AS mn
WHERE NOT EXISTS (
        SELECT 1
        FROM {{vxDBTARGET}} AS mt
        WHERE mt.type = "DD"
            AND mt.docType = "{{vxDOCTYPE}}"
            AND mt.subDocType = "{{vxSUBDOCTYPE}}"
            AND mt.version = "V01"
            AND mt.model = mn
        LIMIT 1
    )
ORDER BY mn;