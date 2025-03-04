package usecase

import (
	"strings"

	"github.com/miu200521358/mlib_go/pkg/mmath"
	"github.com/miu200521358/mlib_go/pkg/mutils/mlog"
	"github.com/miu200521358/mlib_go/pkg/pmx"
	"github.com/miu200521358/mlib_go/pkg/vmd"
	"github.com/miu200521358/mmd-auto-trace-4/pkg/utils"
)

func Reduce(prevMotion *vmd.VmdMotion, modelPath string, moveTolerance, rotTolerance float64, space int, reduceName string, motionNum, allNum int) *vmd.VmdMotion {
	mlog.I("[%d/%d] Reduce ...", motionNum, allNum)

	motion := vmd.NewVmdMotion(strings.Replace(prevMotion.Path, "_heel.vmd", "_fix.vmd", -1))

	minFno := prevMotion.BoneFrames.Get(pmx.CENTER.String()).GetMinFrame()
	maxFno := prevMotion.BoneFrames.Get(pmx.CENTER.String()).GetMaxFrame()
	fnoCounts := maxFno - minFno + 1

	bar := utils.NewProgressBar(fnoCounts * 2)

	// 移動
	moveXs := make(map[string][]float64)
	moveYs := make(map[string][]float64)
	moveZs := make(map[string][]float64)
	for _, boneName := range []string{pmx.CENTER.String(), pmx.LEG_IK.Left(), pmx.LEG_IK.Right()} {
		moveXs[boneName] = make([]float64, fnoCounts)
		moveYs[boneName] = make([]float64, fnoCounts)
		moveZs[boneName] = make([]float64, fnoCounts)
	}

	// 回転
	rots := make(map[string][]float64)
	quats := make(map[string][]*mmath.MQuaternion)
	for boneName := range prevMotion.BoneFrames.Data {
		if boneName != pmx.CENTER.String() {
			rots[boneName] = make([]float64, fnoCounts)
			quats[boneName] = make([]*mmath.MQuaternion, fnoCounts)
		}
	}

	for i := range fnoCounts {
		bar.Increment()
		fno := i + minFno

		for boneName := range prevMotion.BoneFrames.Data {
			if _, ok := moveXs[boneName]; ok {
				bf := prevMotion.BoneFrames.Get(boneName).Get(fno)
				moveXs[boneName][i] = bf.Position.GetX()
				moveYs[boneName][i] = bf.Position.GetY()
				moveZs[boneName][i] = bf.Position.GetZ()
			}
			if _, ok := rots[boneName]; ok {
				bf := prevMotion.BoneFrames.Get(boneName).Get(fno)
				if i == 0 {
					rots[boneName][i] = 1.0
				} else {
					rots[boneName][i] = bf.Rotation.Dot(prevMotion.BoneFrames.Get(boneName).Get(fno - 1).Rotation)
				}
				quats[boneName][i] = bf.Rotation
			}
		}
	}

	moveXInflections := make(map[string]map[int]int)
	moveYInflections := make(map[string]map[int]int)
	moveZInflections := make(map[string]map[int]int)

	for boneName := range moveXs {
		if boneName != pmx.LEG_IK.Left() && boneName != pmx.LEG_IK.Right() {
			moveXInflections[boneName] = mmath.FindInflectionPoints(moveXs[boneName], moveTolerance, space)
			moveYInflections[boneName] = mmath.FindInflectionPoints(moveYs[boneName], moveTolerance, space)
			moveZInflections[boneName] = mmath.FindInflectionPoints(moveZs[boneName], moveTolerance, space)
		} else {
			moveXInflections[boneName] = mmath.FindInflectionPoints(moveXs[boneName], 0.12, space)
			moveYInflections[boneName] = mmath.FindInflectionPoints(moveYs[boneName], 0.12, space)
			moveZInflections[boneName] = mmath.FindInflectionPoints(moveZs[boneName], 0.12, space)
		}
	}

	rotInflections := make(map[string]map[int]int)

	for boneName := range rots {
		if boneName != pmx.LEG_IK.Left() && boneName != pmx.LEG_IK.Right() {
			rotInflections[boneName] = mmath.FindInflectionPoints(rots[boneName], rotTolerance, space)
		} else {
			rotInflections[boneName] = mmath.FindInflectionPoints(rots[boneName], 0.001, space)
		}
	}

	centerXZInflections := mmath.MergeInflectionPoints(moveXs[pmx.CENTER.String()],
		[]map[int]int{moveXInflections[pmx.CENTER.String()], moveZInflections[pmx.CENTER.String()]}, space)
	leftLegIkInflections := mmath.MergeInflectionPoints(moveXs[pmx.LEG_IK.Left()],
		[]map[int]int{moveXInflections[pmx.LEG_IK.Left()], moveYInflections[pmx.LEG_IK.Left()],
			moveZInflections[pmx.LEG_IK.Left()], rotInflections[pmx.LEG_IK.Left()]}, space)
	rightLegIkInflections := mmath.MergeInflectionPoints(moveXs[pmx.LEG_IK.Right()],
		[]map[int]int{moveXInflections[pmx.LEG_IK.Right()], moveYInflections[pmx.LEG_IK.Right()],
			moveZInflections[pmx.LEG_IK.Right()], rotInflections[pmx.LEG_IK.Right()]}, space)

	delete(rotInflections, pmx.LEG_IK.Left())
	delete(rotInflections, pmx.LEG_IK.Right())

	for i := range fnoCounts {
		fno := int(i) + minFno
		bar.Increment()

		if _, ok := centerXZInflections[i]; ok {
			// XZ (センター)
			inflectionIndex := centerXZInflections[i]
			appendCurveFrame(motion, pmx.CENTER.String(), fno, int(inflectionIndex)+minFno,
				moveXs[pmx.CENTER.String()][i:(inflectionIndex+1)], nil, moveZs[pmx.CENTER.String()][i:(inflectionIndex+1)], nil)
		}
		if _, ok := moveYInflections[pmx.CENTER.String()][i]; ok {
			// Y (グルーブ)
			inflectionIndex := moveYInflections[pmx.CENTER.String()][i]
			appendCurveFrame(motion, pmx.GROOVE.String(), fno, int(inflectionIndex)+minFno,
				nil, moveYs[pmx.CENTER.String()][i:(inflectionIndex+1)], nil, nil)
		}
		if _, ok := leftLegIkInflections[i]; ok {
			// 左足IK
			inflectionIndex := leftLegIkInflections[i]
			appendCurveFrame(motion, pmx.LEG_IK.Left(), fno, int(inflectionIndex)+minFno,
				moveXs[pmx.LEG_IK.Left()][i:(inflectionIndex+1)], moveYs[pmx.LEG_IK.Left()][i:(inflectionIndex+1)], moveZs[pmx.LEG_IK.Left()][i:(inflectionIndex+1)],
				quats[pmx.LEG_IK.Left()][i:(inflectionIndex+1)])
		}
		if _, ok := rightLegIkInflections[i]; ok {
			// 右足IK
			inflectionIndex := rightLegIkInflections[i]
			appendCurveFrame(motion, pmx.LEG_IK.Right(), fno, int(inflectionIndex)+minFno,
				moveXs[pmx.LEG_IK.Right()][i:(inflectionIndex+1)], moveYs[pmx.LEG_IK.Right()][i:(inflectionIndex+1)], moveZs[pmx.LEG_IK.Right()][i:(inflectionIndex+1)],
				quats[pmx.LEG_IK.Right()][i:(inflectionIndex+1)])
		}
		for boneName, rotInflection := range rotInflections {
			// 回転ボーン
			if _, ok := rotInflection[i]; ok {
				inflectionIndex := rotInflection[i]
				appendCurveFrame(motion, boneName, fno, int(inflectionIndex)+minFno,
					nil, nil, nil, quats[boneName][i:(inflectionIndex+1)])
			}
		}
	}

	bar.Finish()

	return motion
}

func appendCurveFrame(motion *vmd.VmdMotion, boneName string, startFno, endFno int, xs, ys, zs []float64, quats []*mmath.MQuaternion) {
	startBf := motion.BoneFrames.Get(boneName).Get(startFno)
	endBf := motion.BoneFrames.Get(boneName).Get(endFno)

	if xs != nil && ys == nil && zs != nil {
		startBf.Position = &mmath.MVec3{xs[0], 0, zs[0]}
		endBf.Position = &mmath.MVec3{xs[len(xs)-1], 0, zs[len(zs)-1]}
		endBf.Curves.TranslateX = mmath.NewCurveFromValues(xs)
		endBf.Curves.TranslateZ = mmath.NewCurveFromValues(zs)
	} else if xs == nil && ys != nil && zs == nil {
		startBf.Position = &mmath.MVec3{0, ys[0], 0}
		endBf.Position = &mmath.MVec3{0, ys[len(ys)-1], 0}
		endBf.Curves.TranslateY = mmath.NewCurveFromValues(ys)
	} else if xs != nil && ys != nil && zs != nil {
		startBf.Position = &mmath.MVec3{xs[0], ys[0], zs[0]}
		endBf.Position = &mmath.MVec3{xs[len(xs)-1], ys[len(ys)-1], zs[len(zs)-1]}
		endBf.Curves.TranslateX = mmath.NewCurveFromValues(xs)
		endBf.Curves.TranslateY = mmath.NewCurveFromValues(ys)
		endBf.Curves.TranslateZ = mmath.NewCurveFromValues(zs)
	}

	if quats != nil {
		startBf.Rotation = quats[0]
		endBf.Rotation = quats[len(quats)-1]

		rotTs := make([]float64, len(quats))
		for i, rot := range quats {
			if i == 0 {
				rotTs[i] = 0
			} else if i == len(quats)-1 {
				rotTs[i] = 1
			} else {
				rotTs[i] = mmath.FindSlerpT(quats[0], quats[len(quats)-1], rot)
			}
		}
		endBf.Curves.Rotate = mmath.NewCurveFromValues(rotTs)
	}

	if !motion.BoneFrames.Get(boneName).Contains(startFno) {
		// まだキーフレがない場合のみ開始キーフレ追加
		motion.AppendRegisteredBoneFrame(boneName, startBf)
	}

	// 終端キーフレは補間ありで登録
	motion.AppendRegisteredBoneFrame(boneName, endBf)
}
