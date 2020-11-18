-- snowflake
CREATE SEQUENCE public.global_id_seq;
ALTER SEQUENCE public.global_id_seq OWNER TO postgres;

CREATE OR REPLACE FUNCTION public.sfid()
    RETURNS bigint
    LANGUAGE 'plpgsql'
AS $BODY$
DECLARE
    our_epoch bigint := 1288834974657;
    seq_id bigint;
    now_millis bigint;
    -- the id of this DB shard, must be set for each
    -- schema shard you have - you could pass this as a parameter too
    shard_id int := 1;
    result bigint:= 0;
BEGIN
    SELECT nextval('public.global_id_seq') % 4096 INTO seq_id;

    SELECT FLOOR(EXTRACT(EPOCH FROM clock_timestamp()) * 1000) INTO now_millis;
    result := (now_millis - our_epoch) << 22;
    result := result | (shard_id << 12);
    result := result | (seq_id);
	return result;
END;
$BODY$;

ALTER FUNCTION public.sfid() OWNER TO postgres;

-- SELECT public.sfid();
