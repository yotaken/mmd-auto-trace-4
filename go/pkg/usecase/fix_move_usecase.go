package usecase

import (
	"strings"
	"sync"

	"github.com/cheggaaa/pb/v3"
	"github.com/miu200521358/mlib_go/pkg/mmath"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/mlib_go/pkg/pmx"
	"github.com/miu200521358/mlib_go/pkg/vmd"
	"github.com/miu200521358/mmd-auto-trace-4/pkg/model"
)

func FixMove(allFrames []*model.Frames, allPrevMotions []*vmd.VmdMotion, modelPath string) []*vmd.VmdMotion {
	allMotions := make([]*vmd.VmdMotion, len(allPrevMotions))

	// 全体のタスク数をカウント
	totalFrames := len(allPrevMotions)
	for _, rotMotion := range allPrevMotions {
		totalFrames += int(rotMotion.GetMaxFrame() - rotMotion.GetMinFrame() + 1.0)
	}

	bar := newProgressBar(totalFrames)
	var wg sync.WaitGroup

	for i := range allPrevMotions {
		wg.Add(1)

		go func(i int, frames *model.Frames, prevMotion *vmd.VmdMotion) {
			defer wg.Done()
			motion := fixMoveMotion(frames, prevMotion, bar)

			if mlog.IsVerbose() {
				motion.Path = strings.Replace(allPrevMotions[i].Path, "_ground.vmd", "_fix.vmd", -1)
				err := vmd.Write(motion)
				if err != nil {
					mlog.E("Failed to write leg ik vmd: %v", err)
				}
			}

			allMotions[i] = motion
		}(i, allFrames[i], allPrevMotions[i])
	}

	wg.Wait()
	bar.Finish()

	return allMotions
}

func fixMoveMotion(frames *model.Frames, motion *vmd.VmdMotion, bar *pb.ProgressBar) *vmd.VmdMotion {

	threshold := 0.04
	stopThreshold := threshold * 0.5

	var prevLeftAnklePos2 *mmath.MVec2
	var prevRightAnklePos2 *mmath.MVec2

	for i, fno := range motion.BoneFrames.GetItem("センター").RegisteredIndexes {
		bar.Increment()

		// 2d-jointの足首の位置を取得
		leftAnkleJoint := frames.Frames[int(fno)].Joint2D["OP LAnkle"]
		rightAnkleJoint := frames.Frames[int(fno)].Joint2D["OP RAnkle"]

		if i == 0 {
			prevLeftAnklePos2 = &mmath.MVec2{leftAnkleJoint.X, leftAnkleJoint.Y}
			prevRightAnklePos2 = &mmath.MVec2{rightAnkleJoint.X, rightAnkleJoint.Y}
			continue
		}

		leftAnklePos2 := &mmath.MVec2{leftAnkleJoint.X, leftAnkleJoint.Y}
		rightAnklePos2 := &mmath.MVec2{rightAnkleJoint.X, rightAnkleJoint.Y}

		// 2d-jointの足首の移動量を計算
		leftAnkleDiff2 := leftAnklePos2.Subed(prevLeftAnklePos2)
		rightAnkleDiff2 := rightAnklePos2.Subed(prevRightAnklePos2)

		prevFno := motion.BoneFrames.GetItem("センター").RegisteredIndexes[i-1]

		leftAnkleDiff3 := mmath.NewMVec3()
		rightAnkleDiff3 := mmath.NewMVec3()

		lt := max(0, (threshold-leftAnkleDiff2.Length())/threshold)
		rt := max(0, (threshold-rightAnkleDiff2.Length())/threshold)

		// ほぼ動いていない場合、足IKを止める
		if 0.0 < lt && lt < 1.0 {
			prevLeftAnklePos3 := motion.BoneFrames.Data[pmx.LEG_IK.Left()].Data[prevFno].Position
			nowLeftAnklePos3 := motion.BoneFrames.Data[pmx.LEG_IK.Left()].Data[fno].Position
			leftAnkleDiff3 = prevLeftAnklePos3.Subed(nowLeftAnklePos3)
			leftAnkleDiff3.SetY(0)
			if leftAnkleDiff2.Length() < stopThreshold {
				// 完全停止
				motion.BoneFrames.Data[pmx.LEG_IK.Left()].Data[fno].Position.Add(leftAnkleDiff3)
				mlog.D("[LEFT STOP][%d] prevAnklePos2: %v, AnklePos2: %v, AnkleDiff2: %v, prevAnklePos3: %v, nowAnklePos3: %v, AnkleDiff3: %v",
					int(fno), prevLeftAnklePos2, leftAnklePos2, leftAnkleDiff2.Length(), prevLeftAnklePos3, nowLeftAnklePos3, leftAnkleDiff3)
			} else {
				// 緩やかに停止
				leftAnkleDiff3 = leftAnkleDiff3.MulScalar(lt)
				motion.BoneFrames.Data[pmx.LEG_IK.Left()].Data[fno].Position.Add(leftAnkleDiff3)
				mlog.D("[LEFT MINI STOP][%d] lt: %f, prevAnklePos2: %v, AnklePos2: %v, AnkleDiff2: %v, prevAnklePos3: %v, nowAnklePos3: %v, AnkleDiff3: %v",
					int(fno), lt, prevLeftAnklePos2, leftAnklePos2, leftAnkleDiff2.Length(), prevLeftAnklePos3, nowLeftAnklePos3, leftAnkleDiff3)
			}
		}

		if 0.0 < rt && rt < 1.0 {
			prevRightAnklePos3 := motion.BoneFrames.Data[pmx.LEG_IK.Right()].Data[prevFno].Position
			nowRightAnklePos3 := motion.BoneFrames.Data[pmx.LEG_IK.Right()].Data[fno].Position
			rightAnkleDiff3 = prevRightAnklePos3.Subed(nowRightAnklePos3)
			rightAnkleDiff3.SetY(0)
			if rightAnkleDiff2.Length() < stopThreshold {
				// 完全停止
				motion.BoneFrames.Data[pmx.LEG_IK.Right()].Data[fno].Position.Add(rightAnkleDiff3)
				mlog.D("[RIGHT STOP][%d] prevAnklePos2: %v, AnklePos2: %v, AnkleDiff2: %v, prevAnklePos3: %v, nowAnklePos3: %v, AnkleDiff3: %v",
					int(fno), prevRightAnklePos2, rightAnklePos2, rightAnkleDiff2.Length(), prevRightAnklePos3, nowRightAnklePos3, rightAnkleDiff3)
			} else {
				// 緩やかに停止
				rightAnkleDiff3 = rightAnkleDiff3.MulScalar(rt)
				motion.BoneFrames.Data[pmx.LEG_IK.Right()].Data[fno].Position.Add(rightAnkleDiff3)
				mlog.D("[RIGHT MINI STOP][%d] rt: %f, prevAnklePos2: %v, AnklePos2: %v, AnkleDiff2: %v, prevAnklePos3: %v, nowAnklePos3: %v, AnkleDiff3: %v",
					int(fno), rt, prevRightAnklePos2, rightAnklePos2, rightAnkleDiff2.Length(), prevRightAnklePos3, nowRightAnklePos3, rightAnkleDiff3)
			}
		}

		if (0.0 < lt && lt < 1.0) || (0.0 < rt && rt < 1.0) {
			// センターは両方の差分の平均分移動させる
			ratioLeftAnkleDiff3 := leftAnkleDiff3.MuledScalar(lt)
			ratioRightAnkleDiff3 := rightAnkleDiff3.MuledScalar(rt)
			meanAnkleDiff := ratioLeftAnkleDiff3.Add(ratioRightAnkleDiff3).MulScalar(0.5)
			motion.BoneFrames.Data[pmx.CENTER.String()].Data[fno].Position.Add(meanAnkleDiff)

			mlog.D("[CENTER][%d] leftAnkleDiff3: %v, rightAnkleDiff3: %v, ratioLeftAnkleDiff3: %v, ratioRightAnkleDiff3: %v, meanAnkleDiff: %v",
				int(fno), leftAnkleDiff3, rightAnkleDiff3, ratioLeftAnkleDiff3, ratioRightAnkleDiff3, meanAnkleDiff)
		}
	}

	return motion
}
