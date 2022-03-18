# TODO

- TODO: validate pending txs before node mining
Flow hien tai: Mined succeed => add block => apply txs => Sai value => reject block

- BUGS: 2 node cung mine => cung mined thanh cong 1 thoi diem (<45s sync time) => 2 hash khac nhau

- BUGS: sync time between nodes => 2 node, them pending TXs => thoi gian ticker chay mine tren 2 node khac nhau => mine 2 block khac nhau. Nhung suy nghi cho ki thi k phai bug vi 2 node mine 2 block voi pending TXs khac nhau cung duoc, khi node1 mined succeed => node2 synced block, remove pending TXs tuong ung, huy mine => tiep tuc mine block chua pending TXs con lai [SOLVED]