BEGIN;
DELETE FROM
  "public"."parsed_tx"
WHERE
  "hash" IN (
    '4CD62E3B141DF94B3470730C49FE441C9C4F2462E4DCA085D8F6987A9AB3839B',
    'B467C8115FBE34F5A799740F9A2490E8580AD438A6033743D2453CBFDB0E9114',
    '9CBA5DB9007B9CD8D88987B209BD4B1183792C9805D9AA05442009EA2C15A76E'
  )
  AND "type" = 'initial_provide'
  AND "chain_id" = 'dimension_37-1';
  COMMIT;
