package ai

import "testing"

// 纯内存测试：hub 不碰数据库，零值 Service 即可用（changeSubs 惰性初始化）。
func TestSubscribeChange_DeliversCoalescesAndCancels(t *testing.T) {
	var s Service
	ch, cancel := s.SubscribeChange()

	s.notifyChange()
	select {
	case <-ch:
	default:
		t.Fatal("订阅后 notifyChange 没有送达信号")
	}

	// 连发两次只积一个信号：订阅者的语义是「去重读状态」，不是数事件。
	s.notifyChange()
	s.notifyChange()
	select {
	case <-ch:
	default:
		t.Fatal("合并后的信号丢了")
	}
	select {
	case <-ch:
		t.Fatal("两次 notifyChange 应合并为一个信号, 却收到第二个")
	default:
	}

	cancel()
	cancel() // 幂等
	s.notifyChange()
	select {
	case <-ch:
		t.Fatal("cancel 之后仍在收信号")
	default:
	}
}

func TestSubscribeChange_FansOutToEveryListener(t *testing.T) {
	var s Service
	ch1, cancel1 := s.SubscribeChange()
	ch2, cancel2 := s.SubscribeChange()
	defer cancel2()

	s.notifyChange()
	for i, ch := range []<-chan struct{}{ch1, ch2} {
		select {
		case <-ch:
		default:
			t.Fatalf("订阅者 %d 没收到广播", i+1)
		}
	}

	// 退订一个不影响另一个。
	cancel1()
	s.notifyChange()
	select {
	case <-ch1:
		t.Fatal("已退订的订阅者仍收到广播")
	default:
	}
	select {
	case <-ch2:
	default:
		t.Fatal("另一个订阅者被误伤")
	}
}