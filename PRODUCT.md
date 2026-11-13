# Kipuray

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Go with an embedded React and TypeScript interface, SQLite, Chromium automation, and rrweb. Approved through the product specification and implementation request.

## Users

Web developers, full-stack engineers, and QA working on local applications with synthetic test data.

## Product Purpose

Capture a browser bug as a portable case, reproduce its conditions against local code, and verify a correction with the same expectations before and after.

## Operating Context

A developer's workstation or CI. A separate sharing server stores and displays cases. Capture and execution use a dedicated Chromium profile.

## Capabilities and Constraints

Portable recordings, captured JSON fixtures, response variants, and explicit functional expectations. Execution targets local applications. Fixture-based checks validate frontend behavior, not backend persistence. Supported capture scope is one Chromium tab and conventional DOM interactions.

## Brand Commitments

Kipuray. All product copy and public documentation are in English. The name comes from a Peruvian Southern Quechua vocabulary entry for recording data. No exclusivity or unsupported performance claims.

## Product Principles

- Make the first useful case easy to create.
- Distinguish visual playback from execution against current code.
- Explain missing evidence instead of reporting false success.
- Keep local workflows independent of cloud accounts.
