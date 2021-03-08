import { SigningStargateClient } from "@cosmjs/stargate";
import { Registry } from "@cosmjs/proto-signing";
import { Api } from "./rest";
import { MsgSubmitEvidence } from "./types/cosmos/evidence/v1beta1/tx";
const types = [
    ["/cosmos.evidence.v1beta1.MsgSubmitEvidence", MsgSubmitEvidence],
];
const registry = new Registry(types);
const defaultFee = {
    amount: [],
    gas: "200000",
};
const txClient = async (wallet, { addr: addr } = { addr: "http://localhost:26657" }) => {
    if (!wallet)
        throw new Error("wallet is required");
    const client = await SigningStargateClient.connectWithSigner(addr, wallet, { registry });
    const { address } = (await wallet.getAccounts())[0];
    return {
        signAndBroadcast: (msgs, { fee: fee } = { fee: defaultFee }) => client.signAndBroadcast(address, msgs, fee),
        msgSubmitEvidence: (data) => ({ typeUrl: "/cosmos.evidence.v1beta1.MsgSubmitEvidence", value: data }),
    };
};
const queryClient = async ({ addr: addr } = { addr: "http://localhost:1317" }) => {
    return new Api({ baseUrl: addr });
};
export { txClient, queryClient, };