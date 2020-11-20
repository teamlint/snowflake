-- snowflake
CREATE SEQUENCE public.global_id_seq;
ALTER SEQUENCE public.global_id_seq OWNER TO postgres;

CREATE OR REPLACE FUNCTION public.id()
    RETURNS bigint
    LANGUAGE 'plpgsql'
AS $BODY$
DECLARE
    start_time bigint := 1288834974657;
    seq_id bigint;
    now bigint;
    -- the id of this DB shard, must be set for each
    -- schema shard you have - you could pass this as a parameter too
    node_id int := 1;
    result bigint:= 0;
BEGIN
    SELECT nextval('public.global_id_seq') % 4096 INTO seq_id;

    SELECT FLOOR(EXTRACT(EPOCH FROM clock_timestamp()) * 1000) INTO now;
    result := (now - start_time) << 22;
    result := result | (node_id << 12);
    result := result | (seq_id);
	return result;
END;
$BODY$;

ALTER FUNCTION public.id() OWNER TO postgres;

-- SELECT public.sfid();
