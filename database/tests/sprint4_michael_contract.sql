DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = 'public'
          AND tablename = 'order_stages'
          AND indexname = 'order_stages_one_in_progress_idx'
          AND indexdef LIKE '%WHERE (status = ''IN_PROGRESS''%'
    ) THEN
        RAISE EXCEPTION 'Only one stage may be in progress per order';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'orders' AND column_name = 'cancelled_at'
    ) OR NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'order_stages' AND column_name = 'completed_by'
    ) THEN
        RAISE EXCEPTION 'Order recovery columns are missing';
    END IF;

    IF has_table_privilege('tallerflow_console', 'public.orders', 'SELECT')
       OR has_table_privilege('tallerflow_console', 'public.clients', 'SELECT')
       OR has_table_privilege('tallerflow_console', 'public.stage_photos', 'SELECT') THEN
        RAISE EXCEPTION 'Console role can read workshop content';
    END IF;

    IF NOT has_function_privilege('tallerflow_console', 'public.console_list_workshops(text,integer,integer)', 'EXECUTE')
       OR NOT has_function_privilege('tallerflow_console', 'public.console_set_workshop_status(uuid,text,text,uuid)', 'EXECUTE') THEN
        RAISE EXCEPTION 'Console role cannot use its reviewed functions';
    END IF;
END $$;
