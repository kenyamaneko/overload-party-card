// 利用者が生成物のファイルを直接参照しなくて済むよう、生成された型を本ファイルから再エクスポートする。

import type { components, paths } from "./openapi.gen";

export type { components, paths };

type Schemas = components["schemas"];

/** サービスの死活監視の応答。 */
export type HealthResponse = Schemas["HealthResponse"];
/** カード定義。 */
export type CardDefinition = Schemas["CardDefinition"];
/** プレイヤーの所持状態を付与したカード定義。 */
export type CardWithOwnership = Schemas["CardWithOwnership"];
/** カード定義の項目を付加した、プレイヤーの所持カード。 */
export type PlayerCardWithDef = Schemas["PlayerCardWithDef"];
/** Compute系リソースのステータス。 */
export type ComputeStats = Schemas["ComputeStats"];
/** Data系リソースのステータス。 */
export type DataStats = Schemas["DataStats"];
/** パッシブ効果。 */
export type PassiveEffect = Schemas["PassiveEffect"];
/** プラットフォームの効果。 */
export type PlatformEffect = Schemas["PlatformEffect"];
/** アタッチメントの効果。 */
export type AttachmentEffect = Schemas["AttachmentEffect"];
/** プレイヤーのデッキ。 */
export type Deck = Schemas["Deck"];
/** デッキに含まれるカード 1 種類と枚数。 */
export type DeckCard = Schemas["DeckCard"];
/** デッキの作成・更新リクエストに含める、カード 1 種類と枚数。 */
export type DeckCardEntry = Schemas["DeckCardEntry"];
/** デッキの作成リクエスト。 */
export type DeckCreateRequest = Schemas["DeckCreateRequest"];
/** デッキの更新リクエスト。 */
export type DeckUpdateRequest = Schemas["DeckUpdateRequest"];
/** 陣営に属するプロダクト。 */
export type Product = Schemas["Product"];
/** プロダクトの施策。 */
export type Initiative = Schemas["Initiative"];
/** 施策の区分 (ルーチン / スペシャル)。 */
export type InitiativeKind = Schemas["InitiativeKind"];
/** パッシブ効果の種別。 */
export type PassiveEffectType = Schemas["PassiveEffectType"];
/** プラットフォームの効果の種別。 */
export type PlatformEffectType = Schemas["PlatformEffectType"];
/** アタッチメントの効果の種別。 */
export type AttachmentEffectType = Schemas["AttachmentEffectType"];
