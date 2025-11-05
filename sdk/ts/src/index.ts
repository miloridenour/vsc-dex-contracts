export { VSCDexClient } from './client';
export type { Config, RouteResult, DepositInfo } from './client';

// Re-export types for convenience
export interface PoolInfo {
  id: string;
  asset0: string;
  asset1: string;
  reserve0: number;
  reserve1: number;
  fee: number;
}

export interface TokenInfo {
  symbol: string;
  decimals: number;
  contract_id: string;
  description: string;
}
