#!/usr/bin/env node

/**
 * DEX Router Demo Script
 * Demonstrates complete DEX functionality using the router service
 */

const axios = require('axios');

// Configuration
const ROUTER_URL = process.env.ROUTER_URL || 'http://localhost:8080';
const INDEXER_URL = process.env.INDEXER_URL || 'http://localhost:8081';

class DexDemo {
    constructor() {
        this.client = axios.create({
            baseURL: ROUTER_URL,
            timeout: 10000,
            headers: { 'Content-Type': 'application/json' }
        });
    }

    async apiCall(method, endpoint, data = null) {
        try {
            const config = { method, url: endpoint };
            if (data) config.data = data;
            const response = await this.client.request(config);
            return response.data;
        } catch (error) {
            console.error(`❌ API call failed: ${method} ${endpoint}`);
            console.error(error.response?.data || error.message);
            throw error;
        }
    }

    async delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    async demo() {
        console.log('🚀 Starting DEX Router Demo\n');

        try {
            // Phase 1: Pool Creation
            console.log('📦 Phase 1: Creating Liquidity Pools');

            console.log('Creating HIVE-HBD pool...');
            const pool1 = await this.apiCall('POST', '/api/v1/contract/dex-router/create_pool', {
                asset0: 'HBD',
                asset1: 'HIVE',
                fee_bps: 8
            });
            console.log('✅ HIVE-HBD pool created:', pool1);

            console.log('Creating BTC-HBD pool...');
            const pool2 = await this.apiCall('POST', '/api/v1/contract/dex-router/create_pool', {
                asset0: 'HBD',
                asset1: 'BTC',
                fee_bps: 8
            });
            console.log('✅ BTC-HBD pool created:', pool2);

            // Phase 2: Adding Liquidity
            console.log('\n💰 Phase 2: Adding Liquidity');

            console.log('Adding liquidity to HIVE-HBD pool...');
            const liquidity1 = await this.apiCall('POST', '/api/v1/contract/dex-router/execute', {
                type: 'deposit',
                version: '1.0.0',
                asset_in: 'HBD',
                asset_out: 'HIVE',
                recipient: 'alice',
                metadata: {
                    amount0: '1000000', // 1000 HBD
                    amount1: '500000'    // 500 HIVE
                }
            });
            console.log('✅ Liquidity added to HIVE-HBD pool:', liquidity1);

            console.log('Adding liquidity to BTC-HBD pool...');
            const liquidity2 = await this.apiCall('POST', '/api/v1/contract/dex-router/execute', {
                type: 'deposit',
                version: '1.0.0',
                asset_in: 'HBD',
                asset_out: 'BTC',
                recipient: 'alice',
                metadata: {
                    amount0: '2000000', // 2000 HBD
                    amount1: '100000'    // 1 BTC (100000 sats)
                }
            });
            console.log('✅ Liquidity added to BTC-HBD pool:', liquidity2);

            // Phase 3: Checking Pool State
            console.log('\n📊 Phase 3: Checking Pool State');

            const poolsResponse = await axios.get(`${INDEXER_URL}/indexer/pools`);
            const pools = poolsResponse.data;
            console.log('Current pools:', JSON.stringify(pools, null, 2));

            // Phase 4: Direct Swaps
            console.log('\n🔄 Phase 4: Direct Swaps');

            console.log('Swapping HBD for HIVE...');
            const swap1 = await this.apiCall('POST', '/api/v1/contract/dex-router/execute', {
                type: 'swap',
                version: '1.0.0',
                asset_in: 'HBD',
                asset_out: 'HIVE',
                recipient: 'bob',
                min_amount_out: 245000, // Minimum 245 HIVE
                slippage_bps: 50         // 0.5% slippage tolerance
            });
            console.log('✅ HBD→HIVE swap completed:', swap1);

            console.log('Swapping BTC for HBD...');
            const swap2 = await this.apiCall('POST', '/api/v1/contract/dex-router/execute', {
                type: 'swap',
                version: '1.0.0',
                asset_in: 'BTC',
                asset_out: 'HBD',
                recipient: 'charlie',
                min_amount_out: 19000000, // Minimum 19000 HBD
                slippage_bps: 100          // 1% slippage tolerance
            });
            console.log('✅ BTC→HBD swap completed:', swap2);

            // Phase 5: Two-Hop Swap
            console.log('\n🔀 Phase 5: Two-Hop Swap (BTC → HBD → HIVE)');

            console.log('Swapping BTC for HIVE via HBD...');
            const swap3 = await this.apiCall('POST', '/api/v1/contract/dex-router/execute', {
                type: 'swap',
                version: '1.0.0',
                asset_in: 'BTC',
                asset_out: 'HIVE',
                recipient: 'diana',
                min_amount_out: 95000,   // Minimum 95 HIVE
                slippage_bps: 150         // 1.5% slippage tolerance
            });
            console.log('✅ BTC→HIVE two-hop swap completed:', swap3);

            // Phase 6: Error Handling
            console.log('\n❌ Phase 6: Error Handling Demo');

            console.log('Attempting swap with excessive slippage requirements...');
            try {
                await this.apiCall('POST', '/api/v1/contract/dex-router/execute', {
                    type: 'swap',
                    version: '1.0.0',
                    asset_in: 'BTC',
                    asset_out: 'HIVE',
                    recipient: 'eve',
                    min_amount_out: 1000000000, // Unrealistically high expectation
                    slippage_bps: 1,             // Very tight slippage
                    return_address: {
                        chain: 'BTC',
                        address: 'bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh'
                    }
                });
                console.log('❌ Swap should have failed!');
            } catch (error) {
                console.log('✅ Swap correctly failed with return address handling:', error.response?.data);
            }

            // Phase 7: Referral Fees
            console.log('\n👥 Phase 7: Referral Fees');

            console.log('Swapping with referral...');
            const referralSwap = await this.apiCall('POST', '/api/v1/contract/dex-router/execute', {
                type: 'swap',
                version: '1.0.0',
                asset_in: 'HBD',
                asset_out: 'HIVE',
                recipient: 'frank',
                beneficiary: 'referrer',
                ref_bps: 250,           // 2.5% referral fee
                min_amount_out: 240000
            });
            console.log('✅ Referral swap completed:', referralSwap);

            // Phase 8: Fee Collection
            console.log('\n💸 Phase 8: Fee Collection');

            console.log('Checking accumulated fees...');
            const poolState = await this.apiCall('POST', '/api/v1/contract/dex-router/get_pool', '1');
            console.log('Pool 1 state with fees:', JSON.stringify(poolState, null, 2));

            console.log('Claiming accumulated fees...');
            const feeClaim = await this.apiCall('POST', '/api/v1/contract/dex-router/claim_fees', '1');
            console.log('✅ Fees claimed:', feeClaim);

            // Phase 9: Liquidity Withdrawal
            console.log('\n🏦 Phase 9: Liquidity Withdrawal');

            console.log('Withdrawing partial liquidity...');
            const withdrawal = await this.apiCall('POST', '/api/v1/contract/dex-router/execute', {
                type: 'withdrawal',
                version: '1.0.0',
                asset_in: 'HBD',
                asset_out: 'HIVE',
                recipient: 'alice',
                metadata: {
                    lp_amount: '353553'  // ~50% of LP tokens
                }
            });
            console.log('✅ Partial withdrawal completed:', withdrawal);

            // Final State Check
            console.log('\n📈 Final State Check');
            const finalPools = await axios.get(`${INDEXER_URL}/indexer/pools`);
            console.log('Final pool states:', JSON.stringify(finalPools.data, null, 2));

            console.log('\n🎉 DEX Demo completed successfully!');
            console.log('\nSummary of operations:');
            console.log('✅ 2 pools created');
            console.log('✅ Liquidity added to both pools');
            console.log('✅ 3 successful swaps (1 direct, 2 two-hop)');
            console.log('✅ 1 failed swap with return address');
            console.log('✅ 1 referral swap');
            console.log('✅ Fee collection');
            console.log('✅ Partial liquidity withdrawal');

        } catch (error) {
            console.error('\n💥 Demo failed:', error.message);
            process.exit(1);
        }
    }
}

// Run demo if called directly
if (require.main === module) {
    const demo = new DexDemo();
    demo.demo().catch(console.error);
}

module.exports = DexDemo;
