// data/openapi.yaml から openapi-typescript で生成された components / paths の re-export。
// 利用者は本ファイルが re-export する schema 型を import すること。

import type { components, paths } from "./openapi.gen";

export type { components, paths };

type Schemas = components["schemas"];

export type HealthResponse = Schemas["HealthResponse"];
export type CardDefinition = Schemas["CardDefinition"];
export type CardWithOwnership = Schemas["CardWithOwnership"];
export type PlayerCardWithDef = Schemas["PlayerCardWithDef"];
export type ComputeStats = Schemas["ComputeStats"];
export type DataStats = Schemas["DataStats"];
export type PassiveEffect = Schemas["PassiveEffect"];
export type PlatformEffect = Schemas["PlatformEffect"];
export type AttachmentEffect = Schemas["AttachmentEffect"];
export type Deck = Schemas["Deck"];
export type DeckCard = Schemas["DeckCard"];
export type DeckCardEntry = Schemas["DeckCardEntry"];
export type DeckCreateRequest = Schemas["DeckCreateRequest"];
export type DeckUpdateRequest = Schemas["DeckUpdateRequest"];
export type DeckDetailResponse = Schemas["DeckDetailResponse"];
export type PassiveEffectType = Schemas["PassiveEffectType"];
export type PlatformEffectType = Schemas["PlatformEffectType"];
export type AttachmentEffectType = Schemas["AttachmentEffectType"];
