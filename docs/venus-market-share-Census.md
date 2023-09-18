# EN Venus Market Share Census

## Objective

Measure the QAP market share of SPs running Venus on the Filecoin Mainnet.

## General Idea

Right now, there is no direct way to determine whether a certain SP (Miner Actor) is using Lotus or Venus from on-chain data or from core protocol, that is, it is impossible to obtain complete data of Lotus and Venus SPs, but we can obtain part of the data through to mearure market share of Venus QAP.

##  Data Sampling

### Method: get the miner client used by a certain SP through UserAgent.

*   Desc: Libp2p protocol allows nodes to set UserAgent. Since the default Agents set by Venus and Lotus are different, you can use UserAgent to determine which implementation an SP uses;
    
*   Pre-req: To obtain the UserAgent of an SP, two conditions need to be met:
    
    *   1) The SP has set MultiAddress on the chain;
        
    *   2) The MultiAddress set by the SP is reachable;
        
*   Result: When an SP satisfies the above two conditions, it can be known which implementation the SP uses, and at the same time, we know the MinerID of the SP and all other information (such as QAP) that is associated with the MinerID;
    

###  Steps: Use SPs larger than 10T in the entire network as total, and use the above method to distinguish Venus, Lotus, and Unknown SPs;

01. Obtain the Address of all SPs from on-chain data

02. Screen to retain SPs with QAP greater than 10TiB (total sample)

03. Get MultiAddrs of SP

04. Get SP's UserAgent via MultiAddrs

05. Differentiate Venus SP, Lotus SP, Unknown SP by UserAgent


### Calculation:

#### A. Venus market share census

VenusQAP = Total QAP of Venus SP in total sample

LotusQAP = Total QAP of Lotus SP in total sample

VenusMarketShare = VenusQAP / (VenusQAP + LotusQAP)


### Error analysis

Because only some SPS meet the sample differentiation conditions, that is, full data cannot be obtained, this method is sampling all Venus, Lotus, Unknown SPs, thus the error size depends on how many SPS are Unknown in the whole network. That is, the more SP sets the UserAgent,The smaller the Unknown, the smaller the error.

Up to now (September 2023), there is approximately 10 EiB of QAP available to gain insights into the implementation of its SP through sampling methods. This accounts for almost half of the total QAP on the internet, so it can be considered that the sample space is large enough to meet the requirements of sampling statistics.
