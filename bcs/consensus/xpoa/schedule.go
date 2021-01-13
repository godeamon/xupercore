package xpoa

import (
	"time"

	common "github.com/xuperchain/xupercore/kernel/consensus/base/common"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
)

// xpoaSchedule 实现了ProposerElectionInterface接口，接口定义了validators操作
// xpoaSchedule是xpoa的主要结构，其能通过合约调用来变更smr的候选人信息，并且向smr提供对应round的候选人信息
type xpoaSchedule struct {
	address string
	// 出块间隔, 单位为毫秒
	period int64
	// 每轮每个候选人最多出多少块
	blockNum int64
	// 当前validators的address
	validators []string
	// address到neturl的映射
	addrToNet map[string]string

	ledger    cctx.LedgerRely
	enableBFT bool
}

// minerScheduling 按照时间调度计算目标候选人轮换数term, 目标候选人index和候选人生成block的index
func (s *xpoaSchedule) minerScheduling(timestamp int64, length int) (term int64, pos int64, blockPos int64) {
	// 每一轮的时间
	termTime := s.period * int64(length) * s.blockNum
	// 每个矿工轮值时间
	posTime := s.period * s.blockNum
	term = (timestamp/1e6)/termTime + 1
	//10640483 180000
	resTime := timestamp/1e6 - (term-1)*termTime
	pos = resTime / posTime
	resTime = resTime - (resTime/posTime)*posTime
	blockPos = resTime/s.period + 1
	return
}

// GetLeader 根据输入的round，计算应有的proposer，实现election接口
// 该方法主要为了支撑smr扭转和矿工挖矿，在handleReceivedProposal阶段会调用该方法
// 由于xpoa主逻辑包含回滚逻辑，因此回滚逻辑必须在ProcessProposal进行
// ATTENTION: tipBlock是一个隐式依赖状态
// ATTENTION: 由于GetLeader()永远在GetIntAddress()之前，故在GetLeader时更新schedule的addrToNet Map，可以保证能及时提供Addr到NetUrl的映射
func (s *xpoaSchedule) GetLeader(round int64) string {
	// 若该round已经落盘，则直接返回历史信息，eg. 矿工在当前round的情况
	if b, err := s.ledger.QueryBlockByHeight(round); err == nil {
		return string(b.GetProposer())
	}
	tipBlock := s.ledger.GetTipBlock()
	tipHeight := tipBlock.GetHeight()
	v := s.GetValidators(round)
	if v == nil {
		return ""
	}
	// 计算round对应的timestamp大致区间
	time := time.Now().UnixNano()
	if round > tipHeight {
		time += s.period * 1e6
	}
	_, pos, _ := s.minerScheduling(time, len(v))
	return v[pos]
}

// GetLocalLeader 用于收到一个新块时, 验证该块的时间戳和proposer是否能与本地计算结果匹配
func (s *xpoaSchedule) GetLocalLeader(timestamp int64, round int64) string {
	// xpoa.lg.Info("ConfirmBlock Propcess update validates")
	// ATTENTION: 获取候选人信息时，时刻注意拿取的是check目的round的前三个块，候选人变更是在3个块之后生效，即round-3
	b, err := s.ledger.QueryBlockByHeight(round - 3)
	if err != nil {
		return ""
	}
	localValidators, err := s.getValidatesByBlockId(b.GetBlockid())
	if err != nil {
		return ""
	}
	_, pos, _ := s.minerScheduling(timestamp, len(localValidators))
	return localValidators[pos]
}

// getValidatesByBlockId 根据当前输入blockid，用快照的方式在xmodel中寻找<=当前blockid的最新的候选人值，若无则使用xuper.json中指定的初始值
func (s *xpoaSchedule) getValidatesByBlockId(blockId []byte) ([]string, error) {
	reader, err := s.ledger.CreateSnapshot(blockId)
	if err != nil {
		// xpoa.lg.Error("Xpoa updateValidates getCurrentValidates error", "CreateSnapshot err:", err)
		return nil, err
	}
	res, err := reader.Get(XPOABUCKET, []byte(XPOAKEY))
	if res == nil {
		// 即合约还未被调用，未有变量更新
		return s.validators, nil
	}
	validators, err := common.LoadValidatorsMultiInfo(res.PureData.Value, &s.addrToNet)
	if err != nil {
		return nil, err
	}
	return validators, nil
}

// GetValidators 用于计算目标round候选人信息，同时更新schedule address到internet地址映射
func (s *xpoaSchedule) GetValidators(round int64) []string {
	if round <= 3 {
		return s.validators
	}
	// xpoa的validators变更在包含变更tx的block的后3个块后生效, 即当B0包含了变更tx，在B3时validators才正式统一变更
	tipBlock := s.ledger.GetTipBlock()
	// round区间在(tipBlock()-3, tipBlock()]之间时，validators不会发生改变
	if tipBlock.GetHeight() <= round && round > tipBlock.GetHeight()-3 {
		return s.validators
	}
	b, err := s.ledger.QueryBlockByHeight(round - 3)
	if err != nil {
		// err包含当前高度小于3，s.validators此时是initValidators
		return s.validators
	}
	validators, err := s.getValidatesByBlockId(b.GetBlockid())
	if err != nil {
		return nil
	}
	return validators
}

func (s *xpoaSchedule) GetIntAddress(addr string) string {
	return s.addrToNet[addr]
}

func (s *xpoaSchedule) GetValidatorsMsgAddr() []string {
	var urls []string
	for _, v := range s.validators {
		urls = append(urls, s.addrToNet[v])
	}
	return urls
}

func (s *xpoaSchedule) UpdateValidator() {
	tipBlock := s.ledger.GetTipBlock()
	if tipBlock.GetHeight() <= 3 {
		return
	}
	b, err := s.ledger.QueryBlockByHeight(tipBlock.GetHeight() - 3)
	if err != nil {
		return
	}
	validators, err := s.getValidatesByBlockId(b.GetBlockid())
	if err != nil {
		return
	}
	if !common.AddressEqual(validators, s.validators) {
		s.validators = validators
	}
}