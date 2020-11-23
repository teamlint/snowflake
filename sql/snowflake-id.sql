-- Snowflake
CREATE SEQUENCE public.gid_seq;
ALTER SEQUENCE public.gid_seq OWNER TO postgres;

-- gid_gen(node_id int)
CREATE OR REPLACE FUNCTION public.gid_gen(node_id int)
   RETURNS bigint
   LANGUAGE 'plpgsql'
AS $BODY$
DECLARE
    start_time bigint := 61026175693;
    node_bits  int2 := 10;
    seq_bits   int2 := 10; 

    now		bigint;
    node_id ALIAS FOR $1;
    seq_id	bigint;

    result  bigint:= 0;
BEGIN	
    SELECT nextval('public.gid_seq') % 1024 INTO seq_id;
    SELECT FLOOR(EXTRACT(EPOCH FROM clock_timestamp()) * 1000) INTO now;
    result := (now - start_time) << (node_bits + seq_bits);
    result := result | (node_id << seq_bits);
    result := result | (seq_id);
	return result;
END;
$BODY$;

ALTER FUNCTION public.gid_gen(node_id int) OWNER TO postgres;

-- gid()
CREATE OR REPLACE FUNCTION public.gid()
   RETURNS bigint
   LANGUAGE 'plpgsql'
AS $BODY$
BEGIN	
    return gid_gen(1);
END;
$BODY$;
ALTER FUNCTION public.gid() OWNER TO postgres;

-- SELECT gid(),gid_gen(1);
