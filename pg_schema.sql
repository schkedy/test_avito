--
-- PostgreSQL database dump
--

\restrict zT4zpBHGQ5TCF6pSAbrkzbZfHcsBzNwxckIdXyg0QYC40HAigo5vJdWiIXOxSLK

-- Dumped from database version 16.11
-- Dumped by pg_dump version 16.10 (Ubuntu 16.10-0ubuntu0.24.04.1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: pr_reviewers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.pr_reviewers (
    pull_request_id character varying(255) NOT NULL,
    reviewer_id character varying(255) NOT NULL,
    assigned_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.pr_reviewers OWNER TO postgres;

--
-- Name: pull_requests; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.pull_requests (
    id character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    author_id character varying(255) NOT NULL,
    status character varying(50) NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    merged_at timestamp with time zone,
    CONSTRAINT pull_requests_status_check CHECK (((status)::text = ANY ((ARRAY['OPEN'::character varying, 'MERGED'::character varying])::text[])))
);


ALTER TABLE public.pull_requests OWNER TO postgres;

--
-- Name: teams; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.teams (
    name character varying(255) NOT NULL
);


ALTER TABLE public.teams OWNER TO postgres;

--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    id character varying(255) NOT NULL,
    username character varying(255) NOT NULL,
    team_name character varying(255) NOT NULL,
    is_active boolean DEFAULT true NOT NULL
);


ALTER TABLE public.users OWNER TO postgres;

--
-- Name: pr_reviewers pr_reviewers_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pr_reviewers
    ADD CONSTRAINT pr_reviewers_pkey PRIMARY KEY (pull_request_id, reviewer_id);


--
-- Name: pull_requests pull_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pull_requests
    ADD CONSTRAINT pull_requests_pkey PRIMARY KEY (id);


--
-- Name: teams teams_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT teams_pkey PRIMARY KEY (name);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_pr_author; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_pr_author ON public.pull_requests USING btree (author_id);


--
-- Name: idx_pr_reviewers_reviewer; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_pr_reviewers_reviewer ON public.pr_reviewers USING btree (reviewer_id);


--
-- Name: idx_pr_status; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_pr_status ON public.pull_requests USING btree (status);


--
-- Name: idx_users_team; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_users_team ON public.users USING btree (team_name);


--
-- Name: pr_reviewers pr_reviewers_pull_request_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pr_reviewers
    ADD CONSTRAINT pr_reviewers_pull_request_id_fkey FOREIGN KEY (pull_request_id) REFERENCES public.pull_requests(id) ON DELETE CASCADE;


--
-- Name: pr_reviewers pr_reviewers_reviewer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pr_reviewers
    ADD CONSTRAINT pr_reviewers_reviewer_id_fkey FOREIGN KEY (reviewer_id) REFERENCES public.users(id);


--
-- Name: pull_requests pull_requests_author_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pull_requests
    ADD CONSTRAINT pull_requests_author_id_fkey FOREIGN KEY (author_id) REFERENCES public.users(id);


--
-- Name: users users_team_name_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_team_name_fkey FOREIGN KEY (team_name) REFERENCES public.teams(name) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict zT4zpBHGQ5TCF6pSAbrkzbZfHcsBzNwxckIdXyg0QYC40HAigo5vJdWiIXOxSLK

