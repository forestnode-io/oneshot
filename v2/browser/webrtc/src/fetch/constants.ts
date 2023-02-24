export const DataChannelMTU = 16384;
export const BufferedAmountLowThreshold = 1 * DataChannelMTU; // 2^0 MTU
export const MaxBufferedAmount = 8 * DataChannelMTU; // 2^3 MTU

export const boundary = "boundary";