module Utilities
  def chain
    unless @chain
      @chain = Chain::Client.new
    end
    @chain
  end

  def signer
    unless @signer
      @signer = Chain::HSMSigner.new
    end
    @signer
  end

  def account_balances(account_alias)
    chain.balances.query(filter: "account_alias='#{account_alias}'").reduce({}) do |memo, b|
      key = b.sum_by['asset_alias'].empty? ? b.sum_by['asset_id'] : b.sum_by['asset_alis']
      memo[key] = b.amount
      memo
    end
  end

  def issue(account_alias, asset_alias, amount)
    tx = chain.transactions.build do |b|
      b.issue asset_alias: asset_alias, amount: amount
      b.control_with_account account_alias: account_alias, asset_alias: asset_alias, amount: amount
    end

    chain.transactions.submit(signer.sign(tx))
  end
end
